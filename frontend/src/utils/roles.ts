import { Role } from "../../gen/connect/homecall/v1alpha/tenant_service_pb";

const rolesTranslation = (role: Role) => {
    switch(role) {
        case Role.ADMIN:
            return 'Admin'
        case Role.MEMBER:
            return 'Medlem'
        case Role.UNSPECIFIED:
            return 'Okänd'
        default:
            return 'Okänd'
    }
}

export default rolesTranslation;
