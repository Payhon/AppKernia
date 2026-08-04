# uView Ultra 集成规范

## 1. 固定版本与来源

- 初始固定：`uView Ultra 4.5.18`。
- 来源：DCloud 插件市场正式包。
- 存放：`apps/ak-mobile/uni_modules/uview-ultra`。
- 不使用 npm 模式作为 AK Core 的安装路径。
- 升级必须单独 PR，包含变更日志、组件矩阵、三端截图与回滚方式。

## 2. 官方配置基线

`main.uts`：

```ts
import App from './App.uvue'
import ultraUI from '@/uni_modules/uview-ultra/index.uts'
import { createSSRApp } from 'vue'

export function createApp() {
  const app = createSSRApp(App)
  app.use(ultraUI)
  return { app }
}
```

`uni.scss` 只导入主题：

```scss
@import '@/uni_modules/uview-ultra/theme.scss';
@import '@/design-system/tokens.scss';
```

`App.uvue` 的 style 首行导入基础样式：

```scss
<style lang="scss">
@import '@/uni_modules/uview-ultra/index.scss';
@import '@/design-system/app.scss';
</style>
```

`pages.json`：

```json
{
  "easycom": {
    "autoscan": true,
    "custom": {
      "^up-(.*)": "@/uni_modules/uview-ultra/components/up-$1/up-$1.uvue",
      "^ak-(.*)": "@/components/ak-ui/ak-$1/ak-$1.uvue"
    }
  }
}
```

修改 easycom 后重新编译/HBuilderX 重启，不把热更新结果当作生效证据。

## 3. AK UI 适配层

业务代码：

```vue
<ak-button variant="primary" :loading="saving" @click="submit">
  {{ t('common.save') }}
</ak-button>
```

禁止：

```vue
<up-button type="primary" color="#123456" />
```

AK Wrapper 负责：

- 语义 Token 到 uView prop/SCSS 的映射。
- 平台兼容修正。
- 无障碍文本与触控尺寸。
- Loading/Disabled/Debounce 约定。
- 统一事件和类型。
- 降级到 uni 原生组件。

## 4. 许可

uView Ultra 使用自定义许可。仓库必须：

- 保留完整 Licence 文件。
- 在 `THIRD_PARTY_NOTICES.md` 记录版本和来源。
- 修改第三方源码时在文件中标注，并记录补丁。
- 不把 AK UI 作为 uView Ultra 的竞争性分叉发布。

## 5. 兼容矩阵

`spec/component-compatibility-matrix.json` 是组件准入事实源。

- `approved`：可进入 Core，但仍需三端 Smoke。
- `conditional`：先在 Dev Gallery 完成测试。
- `prohibited_core`：Core 1.0 不使用。

已知“适配中”或风险组件不得绕过矩阵直接进入认证、安全和上传流程。

## 6. 暗色与宽屏

uView Ultra 发布页没有给出内建暗色/宽屏支持保证。因此：

- P0/P1 只承诺手机浅色模式。
- 所有颜色仍使用语义 Token，为后续暗色保留结构。
- `dark_mode` 默认 false；AKMOB-160 通过后才允许打开。
- Pad/横屏只保证不崩溃和关键流程可操作，不承诺桌面化布局。
