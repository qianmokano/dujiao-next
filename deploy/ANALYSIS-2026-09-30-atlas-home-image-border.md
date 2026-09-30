# Atlas 首页商品图片边框调整（2026-09-30）

## 需求与依据

用户要求首页展示的商品图片不加边框。

修改前核对分支、工作区和远端 main，基线为 `800804174e9fd2884858a9d35680eae93c53a42f`，从最新 main 创建 `kano/atlas-home-image-no-border`。原有未提交文件与此次修改不冲突，保留且不暂存。

核对首页两种布局及三处图片实现：`AtlasPlanCard.vue`、`AtlasProductListItem.vue`、`AtlasCategorySidebar.vue`，并参考商品详情图。首页方案卡片和列表模式的商品图片都直接使用 Tailwind `border` 类。

## 实施

移除 `AtlasPlanCard.vue` 和 `AtlasProductListItem.vue` 商品图片上的 `border` 类。图片尺寸、圆角、背景、显示方式、懒加载及加载失败处理沿用原实现；商品卡片、标签和按钮的边框不受影响。

这两个组件仅用于 Atlas 首页，因此无需为其他模板或商品详情增加条件样式。调整仅涉及两处样式类，没有新增公开函数。

## 验证与下一步

- `pnpm run test`：85 项现有测试全部通过。
- `pnpm run build`：Vue 类型检查与 Vite 生产构建通过。
- `git diff --check`：通过，代码差异仅移除两处图片的 `border` 类。此样式调整没有新增业务逻辑，不增加与实现重复的测试。
- 下一步：只提交本次两个组件与本记录，推送功能分支并创建 PR，等待 CI。
