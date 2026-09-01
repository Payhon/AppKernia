import { z } from "zod";
export const feedbackSearchSchema = z.object({
  app_id: z.uuid().optional().catch(undefined),
  q: z.string().max(160).catch(""),
  status: z.enum(["", "pending", "processing", "resolved"]).catch(""),
  created_from: z.iso.datetime({ offset: true }).optional().catch(undefined),
  created_to: z.iso.datetime({ offset: true }).optional().catch(undefined),
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(20),
});
export const replySchema = z.object({ body: z.string().trim().min(1).max(2000), status: z.enum(["pending", "processing", "resolved"]) });
export type FeedbackSearch = z.infer<typeof feedbackSearchSchema>;
export type ReplyForm = z.infer<typeof replySchema>;
export const feedbackStatuses = ["pending", "processing", "resolved"] as const;
