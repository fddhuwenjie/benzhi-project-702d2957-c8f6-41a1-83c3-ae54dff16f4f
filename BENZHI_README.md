# BENZHI_README

## 项目说明
- 项目：benzhi-project-702d2957-c8f6-41a1-83c3-ae54dff16f4f
- 项目用途：已完整实现洁净室环境偏离恢复放行工作台，覆盖影响冻结、结构化调查、纠正措施、连续复测、独立复核、确定性档案及摘要验证，并提供原生浏览器页面和真实 HTTP 全流程自检。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 项目描述
- 项目名称：cleanroom-recovery-ledger
- 项目介绍：面向生物样本制备洁净室的环境偏离恢复放行工作台，将一次微生物或粒子超限事件从基线冻结、影响调查、纠正执行、连续复测推进到独立复核和可验证放行，避免在证据不完整时恢复关键作业。
- 项目概述：面向生物样本制备洁净室的环境偏离恢复放行工作台，将一次微生物或粒子超限事件从基线冻结、影响调查、纠正执行、连续复测推进到独立复核和可验证放行，避免在证据不完整时恢复关键作业。
- 核心工作流：监测员登记洁净室超限偏离并冻结受影响区域和批次时间窗，完成原因调查与纠正措施后提交连续复测结果；系统依据预设门禁判定是否具备复核条件，由未参与处置的质量复核员批准或驳回，批准后生成带摘要的恢复放行档案并冻结案件。
- 对外接口：Go 服务在同一高位回环地址提供原生 HTML、CSS、JavaScript 单页工作台和同源 JSON 接口；页面包含案件列表、状态时间线、调查与纠正表单、复测序列、复核面板及放行档案校验视图，不引入 Node 构建链。服务支持 -addr=127.0.0.1:<port>，默认监听 127.0.0.1:19091，并可在 PORT 为端口号时监听 127.0.0.1:<PORT>，不得默认绑定 0.0.0.0、8080、80 或 3000。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/cleanroom-recovery -self-check -addr=127.0.0.1:19091 -self-check-timeout=12s

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-702d2957-c8f6-41a1-83c3-ae54dff16f4f-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-702d2957-c8f6-41a1-83c3-ae54dff16f4f-arm64 linux/arm64

docker run -it benzhi-project-702d2957-c8f6-41a1-83c3-ae54dff16f4f-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/cleanroom-recovery -self-check -addr=127.0.0.1:19091 -self-check-timeout=12s`
