import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminUserDetailRoute } from '../pages/AdminUserDetailRoute'

export const Route=createFileRoute('/system/users/accounts/$userId')({component:UserDetail})
function UserDetail(){const {userId}=Route.useParams();return <ProtectedPage><AdminUserDetailRoute userId={userId}/></ProtectedPage>}
