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
    }]
  },
]

const router = new VueRouter({
  mode: 'history',
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
  if (!AuthUtils.getToken() && toPath != '/login') {
    next({ path: '/login' });
  } else {
    next();
  }
});

export default router
