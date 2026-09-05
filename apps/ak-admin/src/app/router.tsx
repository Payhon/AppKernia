import { createRouter } from '@tanstack/react-router'

import { routeTree } from '../routeTree.gen'
import { adminBasePath } from './base-path'

export const router = createRouter({ basepath: adminBasePath, routeTree })

declare module '@tanstack/react-router' {
  interface Register { router: typeof router }
}
