<template>
  <div class="active-plate-main">
    <ul class="active-list">
      <li class="item" v-for="item in infoList" :key="item.title">
        <p class="num" :style="{color:item.color}">{{item.count}}</p>
        <p class="desc">{{item.title}}</p>
      </li>
    </ul>
  </div>
</template>

<script>
export default {
  name: 'activePlate',
  components: {},
  props: {
    // 需要展示的数据集合
    infoList: {
      type: Array,
      require: true,
    },
  },
}
</script>

<style lang="scss">
.active-plate-main {
  width: 100%;
  height: 130px;
  .active-list {
    display: flex;
    list-style: none;
    padding-top: 15px;
    .item {
      position: relative;
      flex: 1;
      text-align: center;
      .num {
        font-size: 42px;
        font-weight: bold;
        font-family: sans-serif;
      }
      .desc {
        font-size: 16px;
      }
      &::after {
        position: absolute;
        top: 18px;
        right: 0;
        content: '';
        display: block;
        width: 1px;
        height: 56px;
        background: #e7eef0;
      }
      &:nth-last-of-type(1) {
        &::after {
          background: none;
        }
      }
    }
  }
}
</style>