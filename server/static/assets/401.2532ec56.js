import{J as s,p as l,d as c,e as d,f as m,h as f,i as e,q as _,k as p}from"./vendor.42638b6b.js";import{a as h}from"./index.99723322.js";var t={name:"401",setup(){const n=s();return{onSetAuth:()=>{h(),n.push("/login")}}}},v="assets/401.4efb7617.png",V=`.error[data-v-68c65e35] {
  height: 100%;
  background-color: white;
  display: flex;
}
.error .error-flex[data-v-68c65e35] {
  margin: auto;
  display: flex;
  height: 350px;
  width: 900px;
}
.error .error-flex .left[data-v-68c65e35] {
  flex: 1;
  height: 100%;
  align-items: center;
  display: flex;
}
.error .error-flex .left .left-item .left-item-animation[data-v-68c65e35] {
  opacity: 0;
  animation-name: error-num;
  animation-duration: 0.5s;
  animation-fill-mode: forwards;
}
.error .error-flex .left .left-item .left-item-num[data-v-68c65e35] {
  color: #d6e0f6;
  font-size: 55px;
}
.error .error-flex .left .left-item .left-item-title[data-v-68c65e35] {
  font-size: 20px;
  color: #333333;
  margin: 15px 0 5px 0;
  animation-delay: 0.1s;
}
.error .error-flex .left .left-item .left-item-msg[data-v-68c65e35] {
  color: #c0bebe;
  font-size: 12px;
  margin-bottom: 30px;
  animation-delay: 0.2s;
}
.error .error-flex .left .left-item .left-item-btn[data-v-68c65e35] {
  animation-delay: 0.2s;
}
.error .error-flex .right[data-v-68c65e35] {
  flex: 1;
  opacity: 0;
  animation-name: error-img;
  animation-duration: 2s;
  animation-fill-mode: forwards;
}
.error .error-flex .right img[data-v-68c65e35] {
  width: 100%;
  height: 100%;
}`;const o=_();l("data-v-68c65e35");const x={class:"error"},u={class:"error-flex"},g={class:"left"},y={class:"left-item"},b=e("div",{class:"left-item-animation left-item-num"},"401",-1),w=e("div",{class:"left-item-animation left-item-title"},"\u60A8\u672A\u88AB\u6388\u6743\uFF0C\u6CA1\u6709\u64CD\u4F5C\u6743\u9650",-1),S=e("div",{class:"left-item-animation left-item-msg"},null,-1),k={class:"left-item-animation left-item-btn"},I=p("\u91CD\u65B0\u6388\u6743"),z=e("div",{class:"right"},[e("img",{src:v})],-1);c();const A=o((n,r,C,i,$,j)=>{const a=d("el-button");return m(),f("div",x,[e("div",u,[e("div",g,[e("div",y,[b,w,S,e("div",k,[e(a,{type:"primary",round:"",onClick:i.onSetAuth},{default:o(()=>[I]),_:1},8,["onClick"])])])]),z])])});t.render=A,t.__scopeId="data-v-68c65e35";export default t;
