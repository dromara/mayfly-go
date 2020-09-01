<template>
  <div>
    <template v-for="menu in this.menus">
      <!-- 只有菜单的子节点为菜单类型才继续展开 -->
      <el-submenu
        :key="menu.id"
        :index="!menu.code ? menu.id + '' : menu.code"
        v-if="menu.children && menu.children[0].type === 1"
      >
        <template slot="title">
          <i :class="menu.icon"></i>
          <span slot="title">{{menu.name}}</span>
        </template>
        <MenuTree @toPath="toPath" :menus="menu.children"></MenuTree>
      </el-submenu>
      <el-menu-item
        @click="toPath(menu)"
        :key="menu.id"
        :index="!menu.path ? menu.id + '' : menu.path"
        v-else
      >
        <i class="iconfont" :class="menu.icon"></i>
        <span slot="title">{{menu.name}}</span>
      </el-menu-item>
    </template>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
@Component({
  name: 'MenuTree'
})
export default class MenuTree extends Vue {
  @Prop()
  menus: object

  toPath(menu: any) {
    this.$emit('toPath', menu)
  }
}
</script>>
