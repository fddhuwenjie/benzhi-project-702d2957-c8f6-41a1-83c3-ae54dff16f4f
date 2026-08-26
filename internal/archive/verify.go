package archive

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"cleanroom-recovery-ledger/internal/domain"
)

func Verify(a *domain.ReleaseArchive) Verification {
	return verify(a, a.CaseID, a.SealedRevision, a.ApprovedBy)
}

func VerifyCase(c *domain.DeviationCase) Verification {
	if c == nil || c.Archive == nil {
		return Verification{Valid: false, FailureLocations: []string{"archive"}, Checks: []Check{{Name: "archive", OK: false, Detail: "案件没有放行档案", Expected: "已保存档案", Actual: "无", Location: "archive"}}}
	}
	approvedBy := ""
	if c.Review != nil {
		approvedBy = c.Review.ReviewerID
	}
	return verify(c.Archive, c.CaseID, c.Revision, approvedBy)
}

func verify(a *domain.ReleaseArchive, expectedCaseID string, expectedRevision int64, expectedApprover string) Verification {
	result := Verification{ExpectedDigest: a.CanonicalDigest, Checks: []Check{}, FailureLocations: []string{}}
	m, err := ReadManifest(a.Manifest)
	jsonOK := err == nil
	detail := "规范 JSON 可解析"
	if err != nil {
		detail = err.Error()
	}
	result.add(Check{Name: "manifest_json", OK: jsonOK, Detail: detail, Expected: "可解析 JSON", Actual: choose(jsonOK, "可解析 JSON", detail), Location: "manifest"})
	canonicalOK := false
	if jsonOK {
		again, e := canonicalJSON(m)
		canonicalOK = e == nil && string(again) == string(a.Manifest)
	}
	result.add(Check{Name: "canonical_encoding", OK: canonicalOK, Detail: choose(canonicalOK, "字段与集合顺序符合规范", "保存内容并非规范编码"), Expected: "规范编码", Actual: choose(canonicalOK, "规范编码", "非规范编码"), Location: "manifest"})
	sum := sha256.Sum256(a.Manifest)
	result.ActualDigest = hex.EncodeToString(sum[:])
	digestOK := subtle.ConstantTimeCompare([]byte(result.ActualDigest), []byte(a.CanonicalDigest)) == 1
	result.add(Check{Name: "sha256_digest", OK: digestOK, Detail: choose(digestOK, "SHA-256 摘要一致", "SHA-256 摘要不一致"), Expected: a.CanonicalDigest, Actual: result.ActualDigest, Location: "canonical_digest"})
	actualCaseID := ""
	actualRevision := int64(0)
	actualApprover := ""
	if jsonOK {
		actualCaseID, actualRevision, actualApprover = m.Case.CaseID, m.Case.SealedRevision, m.Approval.ApprovedBy
	}
	result.add(Check{Name: "case_id", OK: jsonOK && actualCaseID == expectedCaseID && a.CaseID == expectedCaseID, Detail: choose(jsonOK && actualCaseID == expectedCaseID && a.CaseID == expectedCaseID, "案件编号一致", "案件编号不一致"), Expected: expectedCaseID, Actual: actualCaseID, Location: "case.case_id"})
	result.add(Check{Name: "sealed_revision", OK: jsonOK && actualRevision == expectedRevision && a.SealedRevision == expectedRevision, Detail: choose(jsonOK && actualRevision == expectedRevision && a.SealedRevision == expectedRevision, "封存修订一致", "封存修订不一致"), Expected: fmt.Sprint(expectedRevision), Actual: fmt.Sprint(actualRevision), Location: "case.sealed_revision"})
	result.add(Check{Name: "approved_by", OK: jsonOK && actualApprover == expectedApprover && a.ApprovedBy == expectedApprover, Detail: choose(jsonOK && actualApprover == expectedApprover && a.ApprovedBy == expectedApprover, "批准人一致", "批准人不一致"), Expected: expectedApprover, Actual: actualApprover, Location: "approval.approved_by"})
	result.Valid = len(result.FailureLocations) == 0
	return result
}

func (v *Verification) add(check Check) {
	v.Checks = append(v.Checks, check)
	if !check.OK {
		v.FailureLocations = append(v.FailureLocations, check.Location)
	}
}
func choose(ok bool, a, b string) string {
	if ok {
		return a
	}
	return b
}
