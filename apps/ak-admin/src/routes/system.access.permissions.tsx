import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminPermissionsRoute } from '../pages/AdminPermissionsRoute'
export const Route=createFileRoute('/system/access/permissions')({component:()=> <ProtectedPage><AdminPermissionsRoute/></ProtectedPage>})
