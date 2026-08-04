import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/system/users/tenants')({ component: Outlet })
