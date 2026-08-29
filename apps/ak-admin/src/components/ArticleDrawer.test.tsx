// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { authSession } from "../features/auth/store";
import type { ArticleInput } from "../features/content/model";
import type { SystemLanguagesState } from "../features/settings/system-languages";
import { LocaleProvider } from "../shared/i18n";
import { ArticleDrawer } from "./ArticleDrawer";

const galleryFileId = "123e4567-e89b-12d3-a456-426614174000";

function languages(): SystemLanguagesState {
  return {
    error: null,
    isError: false,
    isPending: false,
    isReady: true,
    languages: [
      { isDefault: true, label: "简体中文", value: "zh-CN" },
      { isDefault: false, label: "English", value: "en-US" },
    ],
    preferredLocale: "zh-CN",
    refetch: vi.fn(() => Promise.resolve({})),
  };
}

function input(contentType: "gallery" | "video"): ArticleInput {
  return {
    content_type: contentType,
    slug: "existing-content",
    category_ids: [],
    topic_id: null,
    tag_ids: [],
    media: contentType === "gallery" ? [{
      id: "223e4567-e89b-12d3-a456-426614174000",
      file_id: galleryFileId,
      role: "gallery",
      sort_order: 0,
      translations: { "zh-CN": { alt_text: "existing-gallery.jpg" }, "en-US": { alt_text: "existing-gallery.jpg" } },
    }] : [],
    cover_file_id: null,
    reading_minutes: 5,
    allow_comments: true,
    pinned: false,
    featured: false,
    latest: false,
    video_source_type: contentType === "video" ? "external" : null,
    video_file_id: null,
    video_external_url: contentType === "video" ? "https://cdn.example.test/demo.mp4" : null,
    video_duration_seconds: null,
    sort_order: 0,
    translations: {
      "zh-CN": { title: "标题", summary: "摘要", body_format: "markdown", body: "" },
      "en-US": { title: "Title", summary: "Summary", body_format: "markdown", body: "" },
    },
  };
}

function Harness({ contentType }: { contentType: "gallery" | "video" }) {
  const form = useForm<ArticleInput>({ defaultValues: input(contentType) });
  return <ArticleDrawer open fullScreen={false} form={form} categories={[]} topics={[]} tags={[]} languages={languages()} activeLocale="zh-CN" onActiveLocaleChange={vi.fn()} onClose={vi.fn()} onSave={vi.fn()} />;
}

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false, media: query, onchange: null,
      addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
    })),
  });
  vi.stubGlobal("ResizeObserver", class {
    disconnect() { return undefined; }
    observe() { return undefined; }
    unobserve() { return undefined; }
  });
  Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:existing-gallery") });
  Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("ArticleDrawer content media", () => {
  it("separates the cover and labeled options into two responsive rows", () => {
    render(<LocaleProvider><Harness contentType="gallery" /></LocaleProvider>);

    const coverRow = document.querySelector(".ak-content-meta-cover-row");
    const optionsRow = document.querySelector(".ak-content-meta-options-row");
    expect(coverRow).toBeTruthy();
    expect(optionsRow).toBeTruthy();
    expect(coverRow?.textContent).toMatch(/选择封面文件|Choose cover file/);
    expect(optionsRow?.textContent).toMatch(/允许评论|Allow comments/);
    expect(optionsRow?.textContent).toMatch(/置顶|Pinned/);
    expect(optionsRow?.textContent).toMatch(/精选文章|Featured/);
    expect(optionsRow?.textContent).toMatch(/最新|Latest/);
    expect(optionsRow?.textContent).toMatch(/否|No/);
    expect(screen.getAllByRole("switch")).toHaveLength(4);
    expect(screen.getByRole("switch", { name: /允许评论|Allow comments/ })).toBeTruthy();
    expect(screen.getByRole("switch", { name: /置顶|Pinned/ })).toBeTruthy();
    expect(screen.getByRole("switch", { name: /精选文章|Featured/ })).toBeTruthy();
    expect(screen.getByRole("switch", { name: /最新|Latest/ })).toBeTruthy();
  });

  it("loads and displays previews for existing gallery media", async () => {
    const download = vi.spyOn(authSession, "downloadAdminFile").mockResolvedValue({ file: {} as never, blob: new Blob(["image"], { type: "image/jpeg" }) });
    render(<LocaleProvider><Harness contentType="gallery" /></LocaleProvider>);

    fireEvent.click(screen.getByRole("tab", { name: /内容|Content/ }));
    const image = await screen.findByAltText<HTMLImageElement>("existing-gallery.jpg");
    expect(image.src).toContain("blob:existing-gallery");
    expect(download).toHaveBeenCalledWith(galleryFileId);
  });

  it("places video source and external URL fields in the content tab", async () => {
    render(<LocaleProvider><Harness contentType="video" /></LocaleProvider>);

    expect(screen.queryByText(/视频来源|Video source/)).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: /内容|Content/ }));
    await waitFor(() => { expect(screen.getByText(/视频来源|Video source/)).toBeTruthy(); });
    expect(screen.getByDisplayValue("https://cdn.example.test/demo.mp4")).toBeTruthy();
  });
});
