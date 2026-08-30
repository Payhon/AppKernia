# Request

日期：2026-08-28

修复移动端内容查看器测试反馈：

1. 当前自定义基座未包含 `uni-media` 时，视频详情页不得因 `uni.getVideoInfo` 在 mounted hook 中抛错并崩溃。
2. 图文详情的轮播图片点击后进入全屏预览，支持双指缩放与左右切换。
3. 视频从横屏展示切换到竖屏展示后，播放器应在全屏黑色舞台中垂直居中，并避免顶部内容挤压、裁切和大面积无意义空白造成的失衡。

参考截图：`/var/folders/52/mkm9shln40x3hjzr_swl3qfc0000gp/T/codex-clipboard-a41df11b-69ab-4483-85a8-5a0afc24af74.png`。
