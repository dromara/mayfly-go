<template>
    <div>
        <el-dialog title="更换头像" v-model="isShowDialog" width="769px">
            <div class="cropper-warp">
                <div class="cropper-warp-left">
                    <img :src="cropperImg" class="cropper-warp-left-img" />
                </div>
                <div class="cropper-warp-right">
                    <div class="cropper-warp-right-title">预览</div>
                    <div class="cropper-warp-right-item">
                        <div class="cropper-warp-right-value">
                            <img :src="cropperImgBase64" class="cropper-warp-right-value-img" />
                        </div>
                        <div class="cropper-warp-right-label">100 x 100</div>
                    </div>
                    <div class="cropper-warp-right-item">
                        <div class="cropper-warp-right-value">
                            <img :src="cropperImgBase64" class="cropper-warp-right-value-img cropper-size" />
                        </div>
                        <div class="cropper-warp-right-label">50 x 50</div>
                    </div>
                </div>
            </div>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCancel" size="small">取 消</el-button>
                    <el-button type="primary" @click="onSubmit" size="small">更 换</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { reactive, toRefs, nextTick } from 'vue';
import Cropper from 'cropperjs';
import 'cropperjs/dist/cropper.css';
export default {
    name: 'cropperIndex',
    setup() {
        const state = reactive({
            isShowDialog: false,
            cropperImg: '',
            cropperImgBase64: '',
        });
        // 打开弹窗
        const openDialog = (imgs: any) => {
            state.cropperImg = imgs;
            state.isShowDialog = true;
            nextTick(() => {
                initCropper();
            });
        };
        // 关闭弹窗
        const closeDialog = () => {
            state.isShowDialog = false;
        };
        // 取消
        const onCancel = () => {
            closeDialog();
        };
        // 新增
        const onSubmit = () => {};
        // 初始化cropperjs图片裁剪
        const initCropper = () => {
            const letImg: any = document.querySelector('.cropper-warp-left-img');
            const cropper = new Cropper(letImg, {
                viewMode: 1,
                dragMode: 'none',
                initialAspectRatio: 1,
                aspectRatio: 1,
                preview: '.before',
                background: false,
                autoCropArea: 0.6,
                zoomOnWheel: false,
                crop: () => {
                    state.cropperImgBase64 = cropper.getCroppedCanvas().toDataURL('image/jpeg');
                },
            });
        };
        return {
            openDialog,
            closeDialog,
            onCancel,
            onSubmit,
            initCropper,
            ...toRefs(state),
        };
    },
};
</script>

<style scoped lang="scss">
.cropper-warp {
    display: flex;
    .cropper-warp-left {
        position: relative;
        display: inline-block;
        height: 350px;
        flex: 1;
        border: 1px solid #ebeef5;
        background: #fff;
        overflow: hidden;
        background-repeat: no-repeat;
        cursor: move;
        border-radius: 3px;
        .cropper-warp-left-img {
            width: 100%;
            height: 100%;
        }
    }
    .cropper-warp-right {
        width: 150px;
        height: 350px;
        .cropper-warp-right-title {
            text-align: center;
            height: 20px;
            line-height: 20px;
        }
        .cropper-warp-right-item {
            margin: 15px 0;
            .cropper-warp-right-value {
                display: flex;
                .cropper-warp-right-value-img {
                    width: 100px;
                    height: 100px;
                    border-radius: 100%;
                    margin: auto;
                }
                .cropper-size {
                    width: 50px;
                    height: 50px;
                }
            }
            .cropper-warp-right-label {
                text-align: center;
                font-size: 12px;
                color: #666666;
                height: 30px;
                line-height: 30px;
            }
        }
    }
}
</style>
