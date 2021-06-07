import Permission from "../../common/Permission";

export const resourcePermission = {
    resource: Permission.code("resource"),
    save: Permission.code("resource:save"),
    update: Permission.code("resource:update"),
    del: Permission.code("resource:delete")
}

export const rolePermission = {
    role: Permission.code("role"),
    del: Permission.code("role:delete"),
    saveResources: Permission.code("role:saveResources")
}

export const accountPermission = {
    account: Permission.code("account"),
    changeStatus: Permission.code("account:changeStatus"),
    del: Permission.code("account:delete"),
    saveRoles: Permission.code("account:saveRoles")
}
