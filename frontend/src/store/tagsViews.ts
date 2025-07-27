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
    },
});
