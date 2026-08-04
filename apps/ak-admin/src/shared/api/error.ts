import type { ErrorResponse } from '../../generated/api/types.gen'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly messageKey: string
  readonly requestId: string | undefined

  constructor(status: number, response?: ErrorResponse) {
    super(response?.error.message ?? `HTTP ${String(status)}`)
    this.name = 'ApiError'
    this.status = status
    this.code = response?.error.code ?? 'COMMON.UNKNOWN'
    this.messageKey = response?.error.message_key ?? 'errors.common.unknown'
    this.requestId = response?.request_id
  }
}

export async function toApiError(response: Response): Promise<ApiError> {
  let body: ErrorResponse | undefined
  try {
    body = (await response.clone().json()) as ErrorResponse
  } catch {
    body = undefined
  }
  return new ApiError(response.status, body)
}
