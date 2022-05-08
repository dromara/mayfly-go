<template>
    <div>
        <el-form class="search-form" label-position="right" :inline="true">
            <el-form-item prop="project" label="项目" label-width="40px">
                <el-select v-model="projectId" placeholder="请选择项目" @change="changeProject" filterable>
                    <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item prop="env" label="env" label-width="33px">
                <el-select style="width: 80px" v-model="envId" placeholder="环境" @change="changeEnv" filterable>
                    <el-option v-for="item in envs" :key="item.id" :label="item.name" :value="item.id">
                        <span style="float: left">{{ item.name }}</span>
                        <span style="float: right; color: #8492a6; font-size: 13px">{{ item.remark }}</span>
                    </el-option>
                </el-select>
            </el-form-item>

            <slot></slot>
        </el-form>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, defineComponent, onMounted } from 'vue';
import { projectApi } from '../project/api';

export default defineComponent({
    name: 'ProjectEnvSelect',
    props: {
        visible: {
            type: Boolean,
        },
        data: {
            type: Object,
        },
        title: {
            type: String,
        },
        machineId: {
            type: Number,
        },
        isCommon: {
            type: Boolean,
        },
    },
    setup(props: any, { emit }) {
        const state = reactive({
            projects: [] as any,
            envs: [] as any,
            projectId: null,
            envId: null,
        });

        onMounted(async () => {
            state.projects = await projectApi.accountProjects.request(null);
        });

        const changeProject = async (projectId: any) => {
            emit('update:projectId', projectId);
            emit('changeProjectEnv', state.projectId, null);
            state.envId = null;
            state.envs = await projectApi.projectEnvs.request({ projectId });
        };

        const changeEnv = (envId: any) => {
            emit('update:envId', envId);
            emit('changeProjectEnv', state.projectId, envId);
        };

        return {
            ...toRefs(state),
            changeProject,
            changeEnv,
        };
    },
});
</script>
<style lang="scss">
</style>
