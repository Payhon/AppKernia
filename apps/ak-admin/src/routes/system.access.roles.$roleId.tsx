import { createFileRoute } from '@tanstack/react-router'
import { ProtectedPage } from '../app/route-boundaries'
import { AdminRoleDetailRoute } from '../pages/AdminRoleDetailRoute'
export const Route=createFileRoute('/system/access/roles/$roleId')({component:RoleDetail})
function RoleDetail(){const {roleId}=Route.useParams();return <ProtectedPage><AdminRoleDetailRoute roleId={roleId}/></ProtectedPage>}
