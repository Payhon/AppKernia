// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { authSession, useAuthStore } from "../features/auth/store";
import type { AuthContext } from "../features/auth/store";
import { useAdminFiles } from "../features/files/hooks";
import type { AdminFile } from "../generated/api/types.gen";
import { LocaleProvider } from "../shared/i18n";
import { AkFilePicker } from "./AkFilePicker";

const imageFile: AdminFile = { id: "123e4567-e89b-12d3-a456-426614174000", original_name: "article-cover.jpg", media_type: "image/jpeg", extension: "jpg", size_bytes: 17837, provider: "local", status: "ready", scan_status: "skipped", usage_count: 0, created_at: "2026-08-28T00:00:00Z", updated_at: "2026-08-28T00:00:00Z" };
const revokeObjectURLMock = vi.fn();

vi.mock("../features/files/hooks", () => ({
  useAdminFiles: vi.fn(() => ({
    data: { items: [imageFile], total: 1 },
    isError: false,
    isPending: false,
    refetch: vi.fn(() => Promise.resolve()),
  })),
}));

vi.mock("./AkFileUploader", () => ({
  AkFileUploader: () => <div data-testid="file-uploader">Uploader</div>,
}));

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
  Object.defineProperty(HTMLElement.prototype, "setPointerCapture", { configurable: true, value: vi.fn() });
  Object.defineProperty(HTMLElement.prototype, "hasPointerCapture", { configurable: true, value: vi.fn(() => false) });
  Object.defineProperty(HTMLElement.prototype, "releasePointerCapture", { configurable: true, value: vi.fn() });
  Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:file-thumbnail") });
  Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: revokeObjectURLMock });
});

afterEach(() => {
  cleanup();
  useAuthStore.setState({ context: null, status: "anonymous" });
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

describe("AkFilePicker", () => {
  it("renders localized type and preview labels instead of raw translation keys", () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    expect(screen.getByRole("dialog", { name: /选择文件|Choose a file/ })).toBeTruthy();
    expect(screen.getByText(/文件预览|File preview/)).toBeTruthy();
    expect(screen.getByText(/选择文件后可在此处预览|Select a file to preview it here/)).toBeTruthy();
    expect(document.body.textContent).not.toContain("system.files.picker");
  });

  it("loads a protected image thumbnail and places the filename below it", async () => {
    const download = vi.spyOn(authSession, "downloadAdminFile").mockResolvedValue({ file: imageFile, blob: new Blob(["image"], { type: "image/jpeg" }) });
    const view = render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    const thumbnail = await screen.findByRole<HTMLImageElement>("img", { name: "article-cover.jpg" });
    const fileCell = thumbnail.closest(".ak-file-picker-file");
    expect(thumbnail.src).toContain("blob:file-thumbnail");
    expect(fileCell?.firstElementChild?.classList.contains("ak-file-picker-thumb")).toBe(true);
    expect(fileCell?.lastElementChild?.textContent).toBe("article-cover.jpg");
    expect(download).toHaveBeenCalledWith(imageFile.id);

    view.unmount();
    expect(revokeObjectURLMock).toHaveBeenCalledWith("blob:file-thumbnail");
  });

  it("renders compact metadata columns and upload date range filters", () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    expect(screen.getByPlaceholderText(/开始时间|Start date/)).toBeTruthy();
    expect(screen.getByPlaceholderText(/结束时间|End date/)).toBeTruthy();
    expect(screen.getAllByText(/^上传时间$|^Uploaded$/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^大小$|^Size$/).length).toBeGreaterThan(0);
    expect(screen.getByText("17.4 KiB")).toBeTruthy();
    expect(document.querySelector(".ak-file-picker-table .ant-table-small")).toBeTruthy();
    expect(vi.mocked(useAdminFiles)).toHaveBeenLastCalledWith(expect.objectContaining({ created_from: "", created_to: "" }));
  });

  it("switches between grid, compact table, and thumbnail views", async () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    const viewSwitcher = screen.getByRole("button", { name: /切换文件视图|Change file view/ });
    expect(document.querySelector(".ak-file-picker-thumbnail-table")).toBeTruthy();

    fireEvent.click(viewSwitcher);
    fireEvent.click(await screen.findByRole("menuitem", { name: /网格视图|Grid view/ }));
    const gridItem = document.querySelector<HTMLButtonElement>(".ak-file-picker-grid-item");
    expect(gridItem?.textContent).toContain("article-cover.jpg");
    expect(gridItem?.querySelector(".ak-file-picker-grid-meta")?.textContent).toContain("17.4 KiB");

    fireEvent.click(viewSwitcher);
    const tableItems = await screen.findAllByRole("menuitem", { name: /表格视图|Table view/ });
    const tableItem = tableItems.at(-1);
    if (!tableItem) throw new Error("table view menu item was not rendered");
    fireEvent.click(tableItem);
    expect(document.querySelector(".ak-file-picker-compact-table")).toBeTruthy();
    expect(document.querySelector(".ak-file-picker-thumb-table")).toBeTruthy();

    fireEvent.click(viewSwitcher);
    const thumbnailItems = await screen.findAllByRole("menuitem", { name: /缩略图视图|Thumbnail view/ });
    const thumbnailItem = thumbnailItems.at(-1);
    if (!thumbnailItem) throw new Error("thumbnail view menu item was not rendered");
    fireEvent.click(thumbnailItem);
    expect(document.querySelector(".ak-file-picker-thumbnail-table")).toBeTruthy();
  });

  it("places the upload control at the footer left beside the modal actions", () => {
    useAuthStore.setState({ context: { permissions: ["storage.file.upload"] } as AuthContext, status: "authenticated" });
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    const footer = document.querySelector<HTMLElement>(".ak-file-picker-footer");
    const uploadGroup = footer?.querySelector<HTMLElement>(".ak-file-picker-footer-upload");
    const actionGroup = footer?.querySelector<HTMLElement>(".ak-file-picker-footer-actions");
    const uploader = screen.getByTestId("file-uploader");
    expect(footer?.firstElementChild).toBe(uploadGroup);
    expect(footer?.lastElementChild).toBe(actionGroup);
    expect(uploadGroup?.contains(uploader)).toBe(true);
    expect(document.querySelector(".ant-modal-body")?.contains(uploader)).toBe(false);
    if (!actionGroup) throw new Error("file picker footer actions were not rendered");
    expect(within(actionGroup).getByRole("button", { name: /取消|Cancel/ })).toBeTruthy();
    expect(within(actionGroup).getByRole("button", { name: /选择|Select/ })).toBeTruthy();
  });

  it("selects a file by clicking any metadata cell or using the row keyboard action", () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    const sizeCellContent = screen.getByText("17.4 KiB");
    const row = sizeCellContent.closest("tr");
    const selectButton = screen.getByRole<HTMLButtonElement>("button", { name: /选择|Select/ });
    expect(selectButton.disabled).toBe(true);
    expect(row?.tabIndex).toBe(0);
    if (!row) throw new Error("file picker row was not rendered");
    fireEvent.click(sizeCellContent);
    expect(row.getAttribute("aria-selected")).toBe("true");
    expect(row.classList.contains("ant-table-row-selected")).toBe(true);
    expect(selectButton.disabled).toBe(false);
    fireEvent.keyDown(row, { key: " " });
    expect(row.getAttribute("aria-selected")).toBe("true");
    const selectedStatus = screen.getByText(/已选择：article-cover.jpg|Selected: article-cover.jpg/);
    expect(selectedStatus.classList.contains("ak-file-picker-selected")).toBe(true);
    expect(selectedStatus.classList.contains("ak-org-feedback")).toBe(false);
  });

  it("resizes the list and preview, closes and reopens preview, and toggles maximized mode", () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    const onClose = vi.fn();
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={onClose} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    expect(screen.getByRole("separator")).toBeTruthy();
    expect(document.querySelectorAll(".ant-splitter-panel").length).toBe(2);
    fireEvent.click(screen.getByRole("button", { name: /关闭文件预览|Close file preview/ }));
    expect(document.querySelector(".ak-file-picker-preview")).toBeNull();
    const previewOpen = screen.getByRole("button", { name: /展开文件预览|Show file preview/ });
    expect(previewOpen.classList.contains("ak-file-picker-preview-open")).toBe(true);
    expect(previewOpen.textContent).toBe("");
    fireEvent.click(previewOpen);
    expect(document.querySelector(".ak-file-picker-preview")).toBeTruthy();

    const windowActions = document.querySelector<HTMLElement>(".ak-file-picker-window-actions");
    if (!windowActions) throw new Error("window action group was not rendered");
    expect(within(windowActions).getAllByRole("button")).toHaveLength(2);
    fireEvent.click(within(windowActions).getByRole("button", { name: /^关闭$|^Close$/ }));
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: /最大化对话框|Maximize dialog/ }));
    expect(document.querySelector(".ak-file-picker-modal-maximized")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /调整对话框大小|Resize dialog/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /还原对话框|Restore dialog/ }));
    expect(document.querySelector(".ak-file-picker-modal-maximized")).toBeNull();
    expect(screen.getByRole("button", { name: /调整对话框大小|Resize dialog/ })).toBeTruthy();
  }, 15_000);

  it("moves from the title bar and resizes from the bottom-right handle with keyboard controls", () => {
    vi.spyOn(authSession, "downloadAdminFile").mockRejectedValue(new Error("thumbnail unavailable"));
    render(
      <LocaleProvider>
        <AkFilePicker open kind="image" onClose={vi.fn()} onSelect={vi.fn()} />
      </LocaleProvider>,
    );

    const dialog = screen.getByRole("dialog", { name: /选择文件|Choose a file/ });
    const titlebar = document.querySelector<HTMLElement>(".ak-file-picker-titlebar");
    const resizeHandle = screen.getByRole<HTMLButtonElement>("button", { name: /调整对话框大小|Resize dialog/ });
    if (!titlebar) throw new Error("file picker title bar was not rendered");
    const initialTop = dialog.style.top;
    fireEvent.pointerDown(titlebar, { button: 0, pointerId: 1, clientX: 200, clientY: 100 });
    fireEvent.pointerMove(titlebar, { pointerId: 1, clientX: 200, clientY: 108 });
    fireEvent.pointerUp(titlebar, { pointerId: 1, clientX: 200, clientY: 108 });
    expect(dialog.style.top).not.toBe(initialTop);
    const initialWidth = dialog.style.width;
    fireEvent.pointerDown(resizeHandle, { button: 0, pointerId: 2, clientX: 900, clientY: 700 });
    fireEvent.pointerMove(resizeHandle, { pointerId: 2, clientX: 884, clientY: 700 });
    fireEvent.pointerUp(resizeHandle, { pointerId: 2, clientX: 884, clientY: 700 });
    expect(dialog.style.width).not.toBe(initialWidth);
    fireEvent.keyDown(titlebar, { key: "ArrowDown", altKey: true });
    fireEvent.keyDown(resizeHandle, { key: "ArrowLeft" });
  });
});
