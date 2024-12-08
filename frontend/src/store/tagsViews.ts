import { getNowUrl } from '@/common/utils/url';
import { defineStore } from 'pinia';

/**
 * tags view
 */
export const useTagsViews = defineStore('tagsViews', {
    state: (): TagsViewsState => ({
        tagsViews: [],
    }),
    actions: {
        setTagsViews(data: Array<TagsView>) {
            this.tagsViews = data;
        },
        // 设置当前页面的tags view title
        setNowTitle(title: string) {
            this.tagsViews.forEach((item) => {
                // console.log(getNowUrl(), item.path);
                if (item.path == getNowUrl()) {
                    item.title = title;
                }
            });
        },
    },
});
