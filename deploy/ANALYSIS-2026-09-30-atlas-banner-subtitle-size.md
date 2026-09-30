# Atlas Banner 副标题字号调整

## 依据与范围

用户要求副标题字号参考提供的 2048×768 Banner 图片。参考图的副标题约为主标题字号的六成；当前 Atlas 副标题固定 15px，桌面主标题最大 48px，比例偏小。

核对 AtlasBannerHero、VaultBannerHero 和 classic Home 的标题/副标题及响应式写法，从最新 fork main `f7e57737` 创建 `kano/atlas-banner-subtitle-size`。只调整 Atlas Banner 副标题字体大小：手机 18px，桌面使用 `clamp(18px, 2.6vw, 30px)`，在宽屏与 48px 主标题形成接近参考图的比例。保留现有两行截断、换行及主题色处理。

原有未提交文件保留，不修改或暂存。此次视觉样式调整使用现有组件测试与浏览器实际字号/布局检查，不新增重复样式实现的测试。

## 验证与下一步

现有组件 11 项测试通过，行/语句/函数覆盖率 100%、分支 95.45%；类型检查、生产构建和 diff 检查通过。Chromium 在 320/390/767/768/960/1280px 明暗主题及 320/768/1280px 三种语言长文案，共 21 组验证实际字号为 18–30px、约为主标题六成；两行截断有效，按钮无重叠或裁切，没有横向溢出或脚本错误，手机不请求 Banner 背景图。已复核桌面、手机和临界宽度长文案截图。验收脚本及结果保存在本机 `/tmp/atlas-banner-subtitle-size-qa.cjs` 与 `/tmp/atlas-banner-subtitle-size-qa/`。

提交功能分支、创建 PR 并等待 CI；本轮生产发布等待明确指令。
