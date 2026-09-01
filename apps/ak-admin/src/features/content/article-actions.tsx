import { CopyOutlined, DeleteOutlined, EditOutlined, ExportOutlined, MobileOutlined, SendOutlined, StopOutlined } from "@ant-design/icons";
import type { MenuProps } from "antd";
import type { ContentArticle } from "./model";

type Action = "preview" | "view" | "copy" | "edit" | "publish" | "unpublish" | "delete";
export type ArticleActionLabels = Record<Action, string>;
export type ArticleActionHandlers = Record<Action, () => void>;

export function createArticleActionItems(article: Pick<ContentArticle, "status">, permissions: ReadonlySet<string>, appScoped: boolean, url: string | null, labels: ArticleActionLabels, handlers: ArticleActionHandlers, pending: boolean): NonNullable<MenuProps["items"]> {
  const items: NonNullable<MenuProps["items"]> = [];
  const permission = (action: "update" | "publish" | "delete") => permissions.has(`${appScoped ? "app.content" : "content.article"}.${action}`);
  if (article.status === "published" && url) {
    items.push(
      { key: "preview", icon: <MobileOutlined aria-hidden />, label: labels.preview, onClick: handlers.preview },
      { key: "view", icon: <ExportOutlined aria-hidden />, label: labels.view, onClick: handlers.view },
      { key: "copy", icon: <CopyOutlined aria-hidden />, label: labels.copy, onClick: handlers.copy },
    );
  }
  const edits: NonNullable<MenuProps["items"]> = [];
  if (permission("update")) edits.push({ key: "edit", icon: <EditOutlined aria-hidden />, label: labels.edit, disabled: pending, onClick: handlers.edit });
  if (permission("publish") && article.status === "draft") edits.push({ key: "publish", icon: <SendOutlined aria-hidden />, label: labels.publish, disabled: pending, onClick: handlers.publish });
  if (permission("publish") && article.status === "published") edits.push({ key: "unpublish", icon: <StopOutlined aria-hidden />, label: labels.unpublish, disabled: pending, onClick: handlers.unpublish });
  if (edits.length) {
    if (items.length) items.push({ type: "divider" });
    items.push(...edits);
  }
  if (permission("delete")) {
    if (items.length) items.push({ type: "divider" });
    items.push({ key: "delete", icon: <DeleteOutlined aria-hidden />, label: labels.delete, danger: true, disabled: pending, onClick: handlers.delete });
  }
  return items;
}
