import Api from '@/common/Api';

export const indexApi = {
    machineDashbord: Api.newGet('/machines/dashbord'),
    dbDashbord: Api.newGet('/dbs/dashbord'),
    redisDashbord: Api.newGet('/redis/dashbord'),
    mongoDashbord: Api.newGet('/mongos/dashbord'),
};
