export const stableErrorCodes = {
  invalidCredentials: 'IAM.AUTH.INVALID_CREDENTIALS',
  unauthorized: 'AUTH.SESSION.UNAUTHORIZED',
  forbidden: 'ACCESS.PERMISSION.DENIED',
  unknown: 'COMMON.UNKNOWN',
} as const

export type StableErrorCode =
  (typeof stableErrorCodes)[keyof typeof stableErrorCodes]

export function errorTranslationKey(code: string): string {
  return `errors.${code.toLowerCase().replaceAll('_', '-').replaceAll('.', '.')}`
}

