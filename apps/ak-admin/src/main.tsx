import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { router } from './app/router'
import { LocaleProvider } from './shared/i18n'
import { queryClient } from './shared/query-client'
import './styles.css'

const root = document.getElementById('root')
if (!root) throw new Error('ROOT_ELEMENT_MISSING')

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <LocaleProvider><RouterProvider router={router} /></LocaleProvider>
    </QueryClientProvider>
  </StrictMode>,
)
