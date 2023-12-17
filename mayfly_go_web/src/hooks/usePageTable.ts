import Api from '@/common/Api';
import { ElMessage } from 'element-plus';
import { reactive, toRefs } from 'vue';

/**
 * @description table 页面操作方法封装
 * @param {Api} api 获取表格数据 api  (必传)
 * @param {Object} param 获取数据请求参数 (非必传，默认为{pageNum: 1, pageSize: 10})
 * @param {Function} dataCallBack 对api请求返回的数据进行处理的回调方法 (非必传)
 * */
export const usePageTable = (
    api?: Api,
    params: any = {
        // 当前页数
        pageNum: 1,
        // 每页显示条数
        pageSize: 10,
    },
    beforeQueryFn?: (params: any) => any,
    dataCallBack?: (data: any) => any
) => {
    const state = reactive({
        // 表格数据
        tableData: [],
        // 总数量
        total: 0,
        // 查询参数,包含分页参数
        searchParams: params,
        loading: false,
    });

    /**
     * @description 获取表格数据
     * @return void
     * */
    const getTableData = async () => {
        if (!api) return;
        try {
            state.loading = true;
            let sp = { ...state.searchParams };
            if (beforeQueryFn) {
                sp = beforeQueryFn(sp);
                state.searchParams = sp;
            }

            let res = await api.request(sp);
            dataCallBack && (res = dataCallBack(res));

            state.tableData = res.list;
            state.total = res.total;
        } catch (error: any) {
            ElMessage.error(error?.message);
        } finally {
            state.loading = false;
        }
    };

    /**
     * @description 表格数据查询（pageNum = 1）
     * @return void
     * */
    const search = () => {
        state.searchParams.pageNum = 1;
        getTableData();
    };

    /**
     * @description 表格数据重置（pageNum = 1）,除分页相关参数外其他查询参数置为空
     * @return void
     * */
    const reset = () => {
        state.searchParams.pageNum = 1;
        for (let prop of Object.keys(state.searchParams)) {
            if (prop == 'pageNum' || prop == 'pageSize') {
                continue;
            }
            state.searchParams[prop] = null;
        }
        getTableData();
    };

    /**
     * @description 每页条数改变
     * @param {Number} val 当前条数
     * @return void
     * */
    const handlePageSizeChange = (val: number) => {
        state.searchParams.pageNum = 1;
        state.searchParams.pageSize = val;
        getTableData();
    };

    /**
     * @description 当前页改变
     * @param {Number} val 当前页
     * @return void
     * */
    const handlePageNumChange = (val: number) => {
        state.searchParams.pageNum = val;
        getTableData();
    };

    return {
        ...toRefs(state),
        getTableData,
        search,
        reset,
        handlePageSizeChange,
        handlePageNumChange,
    };
};
