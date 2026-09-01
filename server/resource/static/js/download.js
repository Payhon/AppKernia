export function detectPlatform({ userAgent = '', platform = '', maxTouchPoints = 0 } = {}) {
  if (/HarmonyOS|OpenHarmony/i.test(userAgent)) return 'harmony';
  if (/iPhone|iPad|iPod/i.test(userAgent) || (platform === 'MacIntel' && maxTouchPoints > 1)) return 'ios';
  // Vendor identifiers alone cannot distinguish Android from HarmonyOS compatibility UAs.
  if (/HUAWEI|HONOR/i.test(userAgent)) return 'all';
  if (/Android/i.test(userAgent)) return 'android';
  return 'all';
}
export function enhanceDownloads(doc, nav) {
  const group = doc.getElementById('platform-filter');
  if (!group) return;
  const buttons = [...group.querySelectorAll('button')];
  const options = [...doc.querySelectorAll('[data-download-platform]')];
  const status = doc.getElementById('recommendation');
  const select = (platform) => {
    buttons.forEach(button => button.setAttribute('aria-pressed', String(button.dataset.platform === platform)));
    let chosen = false;
    options.forEach(option => {
      const recommended = !chosen && platform !== 'all' && option.dataset.downloadPlatform === platform;
      option.classList.toggle('is-recommended', recommended);
      if (recommended) chosen = true;
    });
    const label = buttons.find(button => button.dataset.platform === platform)?.textContent ?? '';
    status.textContent = chosen ? `${status.dataset.recommended} ${label}` : status.dataset.unknown;
  };
  group.hidden = false;
  buttons.forEach(button => button.addEventListener('click', () => select(button.dataset.platform)));
  select(detectPlatform(nav));
  doc.getElementById('wechat-note').hidden = !/MicroMessenger/i.test(nav.userAgent ?? '');
  const copy = doc.getElementById('copy-link');
  if (copy && nav.clipboard?.writeText) {
    copy.hidden = false;
    copy.addEventListener('click', async () => {
      try { await nav.clipboard.writeText(copy.dataset.url); doc.getElementById('copy-status').textContent = copy.dataset.success; }
      catch { doc.getElementById('copy-status').textContent = copy.dataset.failure; }
    });
  }
}
if (typeof document !== 'undefined') enhanceDownloads(document, navigator);
