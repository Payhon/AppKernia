import { useId, useRef, useState, type KeyboardEvent } from 'react';

type Locale = 'zh-CN' | 'en-US';

type InstallTab = {
  id: 'source' | 'shell' | 'npm' | 'homebrew' | 'release';
  label: string;
  status: string;
  available: boolean;
  description: string;
  command?: string;
  href?: string;
  linkLabel?: string;
  note?: string;
};

const COPY = {
  'zh-CN': {
    label: 'akone 安装方式',
    tabs: [
      {
        id: 'source',
        label: '源码构建',
        status: '当前可用',
        available: true,
        description:
          '需要 Go 1.26.5、Node.js 24 和 pnpm 11。构建产物包含管理端，默认使用 SQLite 启动。',
        command: `git clone https://github.com/Payhon/AppKernia.git
cd AppKernia
corepack enable
pnpm install --frozen-lockfile
make build-akone
mkdir -p "$HOME/.local/bin"
install -m 0755 ./server/bin/akone "$HOME/.local/bin/akone"
export PATH="$HOME/.local/bin:$PATH"
akone version --json`,
        note: 'PATH 修改只作用于当前终端；需要跨会话使用时，请按自己的 Shell 配置持久化。',
      },
      {
        id: 'shell',
        label: 'Shell',
        status: '首个 Preview 后',
        available: false,
        description: '适用于 macOS 和 Linux 的 amd64、arm64；安装器会校验 SHA-256。',
        command: `VERSION='X.Y.Z-preview.N'
curl -fsSL "https://github.com/Payhon/AppKernia/releases/download/v\${VERSION}/install.sh" | sh -s -- --version "\${VERSION}"
export PATH="$HOME/.local/bin:$PATH"
akone version --json`,
        note: '当前下载地址尚未发布。发布后请先从 Release 页复制真实 Preview 版本替换占位值。',
      },
      {
        id: 'npm',
        label: 'npm',
        status: '首个 Preview 后',
        available: false,
        description: '支持 macOS、Linux 和 Windows，通过平台包安装原生 akone 可执行文件。',
        command: `npm install --global @appkernia/akone@preview
akone version --json`,
        note: '当前 npm registry 尚无此包；稳定版发布后可省略 @preview。',
      },
      {
        id: 'homebrew',
        label: 'Homebrew',
        status: '稳定版后',
        available: false,
        description: '适用于 macOS，通过 AppKernia 官方 tap 安装。',
        command: 'brew install payhon/tap/akone',
        note: '当前 tap 尚未发布；发行流程只会为稳定版生成 Formula。',
      },
      {
        id: 'release',
        label: 'Release',
        status: '首个 Preview 后',
        available: false,
        description: '从 GitHub Releases 选择与操作系统和架构匹配的归档，并核对 checksums.txt。',
        command: `akone_<VERSION>_darwin_amd64.tar.gz
akone_<VERSION>_darwin_arm64.tar.gz
akone_<VERSION>_linux_amd64.tar.gz
akone_<VERSION>_linux_arm64.tar.gz
akone_<VERSION>_windows_amd64.zip`,
        href: 'https://github.com/Payhon/AppKernia/releases',
        linkLabel: '查看 GitHub Releases',
        note: '当前还没有公开 Release，因此不提供虚构的版本下载命令。',
      },
    ],
  },
  'en-US': {
    label: 'akone installation methods',
    tabs: [
      {
        id: 'source',
        label: 'Build from source',
        status: 'Available now',
        available: true,
        description:
          'Requires Go 1.26.5, Node.js 24, and pnpm 11. The binary embeds Admin and starts with SQLite by default.',
        command: `git clone https://github.com/Payhon/AppKernia.git
cd AppKernia
corepack enable
pnpm install --frozen-lockfile
make build-akone
mkdir -p "$HOME/.local/bin"
install -m 0755 ./server/bin/akone "$HOME/.local/bin/akone"
export PATH="$HOME/.local/bin:$PATH"
akone version --json`,
        note: 'The PATH change affects only this terminal. Persist it in your shell profile if needed.',
      },
      {
        id: 'shell',
        label: 'Shell',
        status: 'After the first Preview',
        available: false,
        description: 'For macOS and Linux on amd64 or arm64; the installer verifies SHA-256.',
        command: `VERSION='X.Y.Z-preview.N'
curl -fsSL "https://github.com/Payhon/AppKernia/releases/download/v\${VERSION}/install.sh" | sh -s -- --version "\${VERSION}"
export PATH="$HOME/.local/bin:$PATH"
akone version --json`,
        note: 'The download is not published yet. Copy the real Preview version from Releases before running it.',
      },
      {
        id: 'npm',
        label: 'npm',
        status: 'After the first Preview',
        available: false,
        description: 'Installs the native akone executable on macOS, Linux, and Windows.',
        command: `npm install --global @appkernia/akone@preview
akone version --json`,
        note: 'The package is not on the npm registry yet. Omit @preview after a stable release.',
      },
      {
        id: 'homebrew',
        label: 'Homebrew',
        status: 'After a stable release',
        available: false,
        description: 'Installs akone on macOS from the official AppKernia tap.',
        command: 'brew install payhon/tap/akone',
        note: 'The tap is not published yet; the release workflow only creates a Formula for stable releases.',
      },
      {
        id: 'release',
        label: 'Release',
        status: 'After the first Preview',
        available: false,
        description:
          'Choose the archive matching your operating system and architecture on GitHub Releases, then verify checksums.txt.',
        command: `akone_<VERSION>_darwin_amd64.tar.gz
akone_<VERSION>_darwin_arm64.tar.gz
akone_<VERSION>_linux_amd64.tar.gz
akone_<VERSION>_linux_arm64.tar.gz
akone_<VERSION>_windows_amd64.zip`,
        href: 'https://github.com/Payhon/AppKernia/releases',
        linkLabel: 'View GitHub Releases',
        note: 'There is no public Release yet, so no made-up versioned download command is shown.',
      },
    ],
  },
} satisfies Record<Locale, { label: string; tabs: InstallTab[] }>;

function InstallTabContent({ tab }: { tab: InstallTab }) {
  return (
    <>
      <p className="akone-install-tabs__status">
        <span data-available={tab.available}>{tab.status}</span>
      </p>
      <p>{tab.description}</p>
      {tab.command ? (
        <pre className="akone-install-tabs__code">
          <code>{tab.command}</code>
        </pre>
      ) : null}
      {tab.href && tab.linkLabel ? <a href={tab.href}>{tab.linkLabel}</a> : null}
      {tab.note ? <p className="akone-install-tabs__note">{tab.note}</p> : null}
    </>
  );
}

export function AkoneInstallTabs({ locale }: { locale: Locale }) {
  const copy = COPY[locale];
  const tabsId = useId();
  const [activeIndex, setActiveIndex] = useState(0);
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);

  const activateTab = (index: number) => {
    setActiveIndex(index);
    tabRefs.current[index]?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    let nextIndex: number | undefined;

    switch (event.key) {
      case 'ArrowLeft':
        nextIndex = (index - 1 + copy.tabs.length) % copy.tabs.length;
        break;
      case 'ArrowRight':
        nextIndex = (index + 1) % copy.tabs.length;
        break;
      case 'Home':
        nextIndex = 0;
        break;
      case 'End':
        nextIndex = copy.tabs.length - 1;
        break;
      default:
        return;
    }

    event.preventDefault();
    activateTab(nextIndex);
  };

  if (import.meta.env.SSG_MD) {
    return (
      <div className="akone-install-tabs akone-install-tabs--markdown">
        {copy.tabs.map((tab) => (
          <section key={tab.id}>
            <h3>{`${tab.label} · ${tab.status}`}</h3>
            <InstallTabContent tab={tab} />
          </section>
        ))}
      </div>
    );
  }

  return (
    <div className="akone-install-tabs">
      <div aria-label={copy.label} className="akone-install-tabs__list" role="tablist">
        {copy.tabs.map((tab, index) => {
          const selected = index === activeIndex;
          const tabId = `${tabsId}-tab-${tab.id}`;
          const panelId = `${tabsId}-panel-${tab.id}`;

          return (
            <button
              aria-controls={panelId}
              aria-selected={selected}
              className="akone-install-tabs__tab"
              id={tabId}
              key={tab.id}
              onClick={() => setActiveIndex(index)}
              onKeyDown={(event) => handleKeyDown(event, index)}
              ref={(element) => {
                tabRefs.current[index] = element;
              }}
              role="tab"
              tabIndex={selected ? 0 : -1}
              type="button"
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      {copy.tabs.map((tab, index) => {
        const selected = index === activeIndex;

        return (
          <div
            aria-labelledby={`${tabsId}-tab-${tab.id}`}
            className="akone-install-tabs__panel"
            hidden={!selected}
            id={`${tabsId}-panel-${tab.id}`}
            key={tab.id}
            role="tabpanel"
            tabIndex={0}
          >
            <InstallTabContent tab={tab} />
          </div>
        );
      })}
    </div>
  );
}
