import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminMenusRoute } from '../pages/AdminMenusRoute'
export const Route=createFileRoute('/system/access/menus')({component:()=> <ProtectedPage><AdminMenusRoute/></ProtectedPage>})
