import { defineStore } from 'pinia';

/**
 * tags view
 */
export const useTagsViews = defineStore('tagsViews', {
    state: (): TagsViewsState => ({
        tagsViews: [],
        currentRefreshPath: '',
    }),
    actions: {
        setTagsViews(data: Array<TagsView>) {
            this.tagsViews = data;
        },
        setCurrentRefreshPath(path: string) {
            this.currentRefreshPath = path;
        },
    },
});
