# Atlas Banner 素材

- 原始参考：`chatgpt-plus-banner-reference.png`，原图保留，没有覆盖。
- 可用背景：`frontend/user/public/images/atlas/chatgpt-plus-banner-background-v1.png`，1920×720 PNG，版本化文件名。
- 使用内置 imagegen 编辑，仅去掉左侧文字和按钮；保留右侧玻璃图形、光影、配色和左侧留白。工具输出为 2043×770，使用 macOS sips 调整为要求的 1920×720；原始生成输出保留于 Codex generated_images。

## 最终制作提示

```text
Use case: precise-object-edit
Asset type: Atlas subscription storefront desktop banner background
Input images: Image 1 is the edit target.
Primary request: Remove all title text, subtitle text, and the black call-to-action pill button including its text and arrow from the left half. Reconstruct the pale seamless background in those areas. Deliver exactly 1920×720 pixels (8:3) as a PNG.
Constraints: Preserve the original composition, right-side translucent glass ribbon knot, reflective pedestal, sweeping glass surfaces, soft white-blue-lavender-peach palette, light reflections and shadows. Keep the left 58% calm and open for HTML copy. Do not add any typography, icons, buttons, watermarks or objects. Keep all other details as close to the original as possible. Opaque background.
```

## 正式发布时

背景通过前端静态路径 `/images/atlas/chatgpt-plus-banner-background-v1.png` 提供。发布新镜像并确认静态文件可访问后，将后台当前 ChatGPT Plus `home_hero` 横幅的图片地址改为这个路径。保留标题、副标题、商品链接、新窗口设置及排序；不要使用仍含文字的原图，避免重复叠字。

发布前按 deploy/README.md 备份运行状态，并记录当前横幅配置供回退。若独立部署前后端并设置 VITE_API_BASE_URL，应先把背景上传到后台媒体管理，再采用对应上传 URL（现有 getImageUrl 会为相对路径添加 API 基址）。本轮不修改生产横幅配置、不发布镜像或更新 VPS。
