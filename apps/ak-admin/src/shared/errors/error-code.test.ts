import { describe, expect, it } from 'vitest'

import { errorTranslationKey } from './error-code'

describe('errorTranslationKey', () => {
  it('derives a stable local translation key', () => {
    expect(errorTranslationKey('IAM.AUTH.INVALID_CREDENTIALS')).toBe(
      'errors.iam.auth.invalid-credentials',
    )
  })
})

