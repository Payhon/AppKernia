import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminUsersRoute } from '../pages/AdminUsersRoute'

export const Route=createFileRoute('/system/users/accounts/')({component:()=> <ProtectedPage><AdminUsersRoute/></ProtectedPage>})
