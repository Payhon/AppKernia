import { z } from 'zod'

const environmentSchema = z.enum(['development', 'test', 'production'])

export const adminEnvironmentSchema = z
  .object({
    VITE_AK_API_BASE_URL: z
      .string()
      .refine((value) => value === '/admin-api/v1' || (URL.canParse(value) && value.endsWith('/admin-api/v1')), {
        error: 'validation.api_base_url.admin_prefix',
      }),
    VITE_AK_APP_ENV: environmentSchema,
  })
  .readonly()

export type AdminEnvironment = z.infer<typeof adminEnvironmentSchema>

export function parseAdminEnvironment(
  values: Readonly<Record<string, unknown>>,
): AdminEnvironment {
  return adminEnvironmentSchema.parse(values)
}
