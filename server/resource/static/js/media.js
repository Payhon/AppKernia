const clamp = (value, minimum, maximum) => Math.max(minimum, Math.min(maximum, value));

for (const gallery of document.querySelectorAll("[data-gallery]")) {
  const track = gallery.querySelector("[data-gallery-track]");
  const slides = Array.from(gallery.querySelectorAll("[data-gallery-slide]"));
  const counter = gallery.querySelector("[data-gallery-counter]");
  const thumbnails = Array.from(gallery.querySelectorAll("[data-gallery-thumbnail]"));
  if (!(track instanceof HTMLElement) || slides.length < 2) continue;
  let index = 0;
  let frame = 0;
  const update = (next) => {
    index = clamp(next, 0, slides.length - 1);
    if (counter) counter.textContent = `${index + 1} / ${slides.length}`;
    thumbnails.forEach((thumbnail, position) => {
      if (position === index) thumbnail.setAttribute("aria-current", "true");
      else thumbnail.removeAttribute("aria-current");
    });
  };
  const move = (next) => {
    update(next);
    slides[index]?.scrollIntoView({ behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "nearest", inline: "center" });
  };
  gallery.querySelector("[data-gallery-previous]")?.addEventListener("click", () => move(index - 1));
  gallery.querySelector("[data-gallery-next]")?.addEventListener("click", () => move(index + 1));
  thumbnails.forEach((thumbnail, position) => thumbnail.addEventListener("click", () => move(position)));
  track.addEventListener("scroll", () => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => update(Math.round(track.scrollLeft / Math.max(track.clientWidth, 1))));
  }, { passive: true });
}

for (const viewer of document.querySelectorAll("[data-video-viewer]")) {
  const video = viewer.querySelector("video");
  const buttons = Array.from(viewer.querySelectorAll("[data-video-mode]"));
  let selected = false;
  const setMode = (mode, userSelected = true) => {
    selected ||= userSelected;
    viewer.setAttribute("data-mode", mode);
    buttons.forEach((button) => button.setAttribute("aria-pressed", String(button.getAttribute("data-video-mode") === mode)));
    if (userSelected && video instanceof HTMLVideoElement) video.pause();
  };
  buttons.forEach((button) => button.addEventListener("click", () => setMode(button.getAttribute("data-video-mode") === "vertical" ? "vertical" : "horizontal")));
  if (video instanceof HTMLVideoElement) video.addEventListener("loadedmetadata", () => {
    if (!selected && video.videoHeight > video.videoWidth) setMode("vertical", false);
  }, { once: true });
}
