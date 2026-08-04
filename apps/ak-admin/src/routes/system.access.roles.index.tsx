import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminRolesRoute } from '../pages/AdminRolesRoute'
export const Route=createFileRoute('/system/access/roles/')({component:()=> <ProtectedPage><AdminRolesRoute/></ProtectedPage>})
