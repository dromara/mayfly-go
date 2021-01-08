<template>
  <div class="main">
    <div class="header">
      <div class="logo">
        <span class="big">Mayfly-Go</span>
      </div>
      <div class="right">
        <span class="header-btn">
          <el-badge :value="3" class="badge">
            <i class="el-icon-bell"></i>
          </el-badge>
        </span>
        <el-dropdown>
          <span class="header-btn">
            {{username}}
            <i class="el-icon-arrow-down el-icon--right"></i>
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item @click.native="this.$router.push('/personal')">
              <i style="padding-right: 8px" class="fa fa-cog"></i>个人中心
            </el-dropdown-item>
            <el-dropdown-item @click.native="logout">
              <i style="padding-right: 8px" class="fa fa-key"></i>退出系统
            </el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </div>
    </div>

    <div class="app">
      <div class="aside">
        <div class="menu">
          <el-menu
            background-color="#222d32"
            text-color="#bbbbbb"
            active-text-color="#fff"
            class="menu"
          >
            <MenuTree @toPath="toPath" :menus="this.menus"></MenuTree>
          </el-menu>
        </div>
      </div>
      <div class="app-body">
        <el-tabs
          id="nav-bar"
          class="none-select"
          v-model="activeName"
          @tab-click="tabClick"
          @tab-remove="removeTab"
          type="card"
          closable
        >
          <el-tab-pane
            :key="item.name"
            v-for="(item) in tabs"
            :label="item.title"
            :name="item.name"
          ></el-tab-pane>
        </el-tabs>
        <div id="mainContainer" class="main-container">
          <router-view v-if="!iframe"></router-view>
          <iframe
            style="width: calc(100% - 235px); height: calc(100% - 90px)"
            rameborder="0"
            v-else
            :src="iframeSrc"
          ></iframe>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'
import MenuTree from './MenuTree.vue'
import api from '@/common/openApi'
import { AuthUtils } from '../common/AuthUtils'

@Component({
  name: 'Layout',
  components: {
    MenuTree,
  },
})
export default class App extends Vue {
  private iframe = false
  private iframeSrc: string | null = null
  private username = ''
  private menus: Array<object> = []
  private tabs: Array<any> = []
  private activeName = ''
  private tabIndex = 2

  private toPath(menu: any) {
    const path = menu.url
    this.goToPath(path)
    this.addTab(path, menu.name)
  }

  private goToPath(path: string) {
    // 如果是请求其他地址，则使用iframe展示
    if (path && (path.startsWith('http://') || path.startsWith('https://'))) {
      this.iframe = true
      this.iframeSrc = path
      return
    }
    this.iframe = false
    this.iframeSrc = null
    this.$router
      .push({
        path,
      })
      // eslint-disable-next-line @typescript-eslint/no-empty-function
      .catch((err: any) => {})
  }

  private tabClick(tab: any) {
    this.goToPath(tab.name)
  }

  private addTab(path: string, title: string) {
    for (const n of this.tabs) {
      if (n.name === path) {
        this.activeName = path
        return
      }
    }
    this.tabs.push({
      name: path,
      title: title,
    })
    this.activeName = path
  }

  private removeTab(targetName: string) {
    const tabs = this.tabs
    let activeName = this.activeName
    if (activeName === targetName) {
      tabs.forEach((tab, index) => {
        if (tab.name == targetName) {
          const nextTab = tabs[index + 1] || tabs[index - 1]
          if (nextTab) {
            activeName = nextTab.name
          }
        }
      })
    }
    this.activeName = activeName
    this.tabs = tabs.filter((tab) => tab.name !== targetName)
    this.goToPath(activeName)
  }

  private async logout() {
    sessionStorage.clear()
    this.$router.push({
      path: '/login',
    })
  }

  mounted() {
    const menu = [
      {
        id: 1,
        type: 1,
        name: '机器管理',
        icon: 'el-icon-menu',
        children: [
          {
            id: 11,
            type: 1,
            name: '机器列表',
            url: '/machines',
            icon: 'el-icon-menu',
            code: 'machines',
          },
        ],
      },
      {
        id: 2,
        type: 1,
        name: 'DBMS',
        icon: 'el-icon-menu',
        children: [
          {
            id: 21,
            type: 1,
            name: '数据查询',
            url: '/db-select-data',
            icon: 'el-icon-menu',
            code: 'db-select',
          },
        ],
      },
    ]

    if (menu != null) {
      this.menus = menu
    }

    const user = sessionStorage.getItem('admin')
    if (user != null) {
      this.username = JSON.parse(user).username
    }

    this.addTab(this.$route.path, this.$route.meta.title)
  }
}
</script>>
<style lang="less">
.main {
  display: flex;

  .el-menu:not(.el-menu--collapse) {
    width: 230px;
  }

  .app {
    width: 100%;
    background-color: #ecf0f5;
  }

  .aside {
    position: fixed;
    margin-top: 50px;
    z-index: 10;
    background-color: #222d32;
    transition: all 0.3s ease-in-out;

    .menu {
      overflow-y: auto;
      height: calc(~'100vh');
    }
  }

  .app-body {
    margin-left: 230px;
    -webkit-transition: margin-left 0.3s ease-in-out;
    transition: margin-left 0.3s ease-in-out;
  }

  .main-container {
    margin-top: 88px;
    padding: 2px;
    min-height: calc(~'100vh - 88px');
  }
}

.header {
  width: 100%;
  position: fixed;
  display: flex;
  height: 50px;
  background-color: #303643;
  z-index: 10;

  .logo {
    .min {
      display: none;
    }

    width: 230px;
    height: 50px;
    text-align: center;
    line-height: 50px;
    color: #fff;
    background-color: #303643;
    -webkit-transition: width 0.35s;
    transition: all 0.3s ease-in-out;
  }

  .right {
    position: absolute;
    right: 0;
  }

  .header-btn {
    .el-badge__content {
      top: 14px;
      right: 7px;
      text-align: center;
      font-size: 9px;
      padding: 0 3px;
      background-color: #00a65a;
      color: #fff;
      border: none;
      white-space: nowrap;
      vertical-align: baseline;
      border-radius: 0.25em;
    }

    overflow: hidden;
    height: 50px;
    display: inline-block;
    text-align: center;
    line-height: 50px;
    cursor: pointer;
    padding: 0 14px;
    color: #fff;

    &:hover {
      background-color: #222d32;
    }
  }
}

.menu {
  border-right: none;
  // 禁止选择
  -moz-user-select: -moz-none;
  -moz-user-select: none;
  -o-user-select: none;
  -khtml-user-select: none;
  -webkit-user-select: none;
  -ms-user-select: none;
  user-select: none;
}

.el-menu--vertical {
  min-width: 190px;
}

.setting-category {
  padding: 10px 0;
  border-bottom: 1px solid #eee;
}

#mainContainer iframe {
  border: none;
  outline: none;
  width: 100%;
  height: 100%;
  position: absolute;
  background-color: #ecf0f5;
}

.el-submenu__title {
  font-weight: 500;
}
.el-menu-item {
  font-weight: 500;
}

#nav-bar {
  margin-top: 50px;
  height: 38px;
  width: 100%;
  z-index: 8;
  background: #fff;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.12), 0 0 3px 0 rgba(0, 0, 0, 0.04);
  position: fixed;
  top: 0;
}
</style>
