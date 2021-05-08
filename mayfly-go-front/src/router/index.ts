import Vue from 'vue'
import VueRouter, { RouteConfig } from 'vue-router'
import Layout from "@/layout/Layout.vue"
import { AuthUtils } from '../common/AuthUtils';

Vue.use(VueRouter)

const routes: Array<RouteConfig> = [
  {
    path: '/login',
    name: 'Login',
    meta: {
      title: '登录',
      keepAlive: false
    },
    component: () => import('@/views/login/Login.vue')
  },
  {
    path: '/',
    component: Layout,
    meta: {
      title: '首页',
      keepAlive: false,
    },
    children: [{
      path: 'machines',
      name: 'machines',
      meta: {
        title: '机器列表',
        keepAlive: false
      },
      component: () => import('@/views/machine')
    },
    {
      path: 'db-select-data',
      name: 'dbs',
      meta: {
        title: 'DBMS',
        keepAlive: false
      },
      component: () => import('@/views/db/SelectData.vue')
      // children: [{
      //   path: 'select',
      //   name: 'select-data',
      //   meta: {
      //     title: 'DBMS',
      //     keepAlive: false
      //   },
      //   component: () => import('@/views/db/SelectData.vue')
      // }]
    },
    {
      path: 'mock-data',
      name: 'mock-data',
      meta: {
        title: '数据列表',
        keepAlive: false
      },
      component: () => import('@/views/mock-server')
    },
    ]
  },
  {
    path: '/machines/:id/terminal',
    name: 'machine-terminal',
    meta: {
      title: '终端',
      keepAlive: false
    },
    component: () => import('@/views/machine/SshTerminalPage.vue')
  },
  
]

const router = new VueRouter({
  // hash模式可解决部署服务器后，刷新页面404问题
  mode: 'hash',
  base: process.env.BASE_URL,
  routes
})

router.beforeEach((to: any, from: any, next: any) => {
  window.document.title = to.meta.title
  const toPath = to.path
  if (toPath.startsWith('/open')) {
    next()
    return
  }
  // if (!AuthUtils.getToken() && toPath != '/login') {
  //   next({ path: '/login' });
  // } else {
  //   next();
  // }

  next();
});

export default router
