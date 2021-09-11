import Api from '@/common/Api';

export const indexApi = {
    getIndexCount: Api.create("/common/index/count", 'get'),
}

