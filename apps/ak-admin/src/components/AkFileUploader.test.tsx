// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LocaleProvider } from "../shared/i18n";
import { AkFileUploader } from "./AkFileUploader";

vi.mock("../features/files/hooks", () => ({
  useAdminFileUploadPolicy: () => ({
    data: {
      configuration_safe: true,
      file_media_types: ["image/jpeg"],
      max_file_bytes: 104857600,
      provider: "local",
    },
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("../features/settings/hooks", () => ({
  useAdminDictionary: () => ({ data: { items: [{ label: "Local", value: "local" }] } }),
}));

afterEach(cleanup);

describe("AkFileUploader", () => {
  it("shows an upload icon beside the upload action label", () => {
    render(
      <LocaleProvider>
        <AkFileUploader compact />
      </LocaleProvider>,
    );

    const uploadButton = screen.getByRole("button", { name: /上传文件|Upload file/ });
    expect(uploadButton.querySelector('svg[data-icon="upload"]')).toBeTruthy();
  });
});
