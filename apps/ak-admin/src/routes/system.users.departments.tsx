import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { OrgDepartmentsRoute } from '../pages/OrgDepartmentsRoute'

export const Route = createFileRoute('/system/users/departments')({ component: () => <ProtectedPage><OrgDepartmentsRoute /></ProtectedPage> })
