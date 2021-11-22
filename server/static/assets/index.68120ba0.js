var C=Object.defineProperty;var y=Object.getOwnPropertySymbols;var D=Object.prototype.hasOwnProperty,B=Object.prototype.propertyIsEnumerable;var F=(i,t,n)=>t in i?C(i,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):i[t]=n,E=(i,t)=>{for(var n in t||(t={}))D.call(t,n)&&F(i,n,t[n]);if(y)for(var n of y(t))B.call(t,n)&&F(i,n,t[n]);return i};import{u as I,f as T}from"./index.01696ebf.js";import{A as k}from"./Api.7190d43f.js";import{a as S,y as A,o as j,t as z,n as M,p as U,d as q,e as _,f as g,h as x,i as s,l as w,F as L,E as O,q as P}from"./vendor.c08e96cf.js";var b=function(){return(b=Object.assign||function(i){for(var t,n=1,m=arguments.length;n<m;n++)for(var e in t=arguments[n])Object.prototype.hasOwnProperty.call(t,e)&&(i[e]=t[e]);return i}).apply(this,arguments)},c=function(){function i(t,n,m){var e=this;this.target=t,this.endVal=n,this.options=m,this.version="2.0.7",this.defaults={startVal:0,decimalPlaces:0,duration:2,useEasing:!0,useGrouping:!0,smartEasingThreshold:999,smartEasingAmount:333,separator:",",decimal:".",prefix:"",suffix:""},this.finalEndVal=null,this.useEasing=!0,this.countDown=!1,this.error="",this.startVal=0,this.paused=!0,this.count=function(r){e.startTime||(e.startTime=r);var o=r-e.startTime;e.remaining=e.duration-o,e.useEasing?e.countDown?e.frameVal=e.startVal-e.easingFn(o,0,e.startVal-e.endVal,e.duration):e.frameVal=e.easingFn(o,e.startVal,e.endVal-e.startVal,e.duration):e.countDown?e.frameVal=e.startVal-(e.startVal-e.endVal)*(o/e.duration):e.frameVal=e.startVal+(e.endVal-e.startVal)*(o/e.duration),e.countDown?e.frameVal=e.frameVal<e.endVal?e.endVal:e.frameVal:e.frameVal=e.frameVal>e.endVal?e.endVal:e.frameVal,e.frameVal=Number(e.frameVal.toFixed(e.options.decimalPlaces)),e.printValue(e.frameVal),o<e.duration?e.rAF=requestAnimationFrame(e.count):e.finalEndVal!==null?e.update(e.finalEndVal):e.callback&&e.callback()},this.formatNumber=function(r){var o,l,a,h,d,N=r<0?"-":"";if(o=Math.abs(r).toFixed(e.options.decimalPlaces),a=(l=(o+="").split("."))[0],h=l.length>1?e.options.decimal+l[1]:"",e.options.useGrouping){d="";for(var u=0,V=a.length;u<V;++u)u!==0&&u%3==0&&(d=e.options.separator+d),d=a[V-u-1]+d;a=d}return e.options.numerals&&e.options.numerals.length&&(a=a.replace(/[0-9]/g,function(f){return e.options.numerals[+f]}),h=h.replace(/[0-9]/g,function(f){return e.options.numerals[+f]})),N+e.options.prefix+a+h+e.options.suffix},this.easeOutExpo=function(r,o,l,a){return l*(1-Math.pow(2,-10*r/a))*1024/1023+o},this.options=b(b({},this.defaults),m),this.formattingFn=this.options.formattingFn?this.options.formattingFn:this.formatNumber,this.easingFn=this.options.easingFn?this.options.easingFn:this.easeOutExpo,this.startVal=this.validateValue(this.options.startVal),this.frameVal=this.startVal,this.endVal=this.validateValue(n),this.options.decimalPlaces=Math.max(this.options.decimalPlaces),this.resetDuration(),this.options.separator=String(this.options.separator),this.useEasing=this.options.useEasing,this.options.separator===""&&(this.options.useGrouping=!1),this.el=typeof t=="string"?document.getElementById(t):t,this.el?this.printValue(this.startVal):this.error="[CountUp] target is null or undefined"}return i.prototype.determineDirectionAndSmartEasing=function(){var t=this.finalEndVal?this.finalEndVal:this.endVal;this.countDown=this.startVal>t;var n=t-this.startVal;if(Math.abs(n)>this.options.smartEasingThreshold){this.finalEndVal=t;var m=this.countDown?1:-1;this.endVal=t+m*this.options.smartEasingAmount,this.duration=this.duration/2}else this.endVal=t,this.finalEndVal=null;this.finalEndVal?this.useEasing=!1:this.useEasing=this.options.useEasing},i.prototype.start=function(t){this.error||(this.callback=t,this.duration>0?(this.determineDirectionAndSmartEasing(),this.paused=!1,this.rAF=requestAnimationFrame(this.count)):this.printValue(this.endVal))},i.prototype.pauseResume=function(){this.paused?(this.startTime=null,this.duration=this.remaining,this.startVal=this.frameVal,this.determineDirectionAndSmartEasing(),this.rAF=requestAnimationFrame(this.count)):cancelAnimationFrame(this.rAF),this.paused=!this.paused},i.prototype.reset=function(){cancelAnimationFrame(this.rAF),this.paused=!0,this.resetDuration(),this.startVal=this.validateValue(this.options.startVal),this.frameVal=this.startVal,this.printValue(this.startVal)},i.prototype.update=function(t){cancelAnimationFrame(this.rAF),this.startTime=null,this.endVal=this.validateValue(t),this.endVal!==this.frameVal&&(this.startVal=this.frameVal,this.finalEndVal||this.resetDuration(),this.finalEndVal=null,this.determineDirectionAndSmartEasing(),this.rAF=requestAnimationFrame(this.count))},i.prototype.printValue=function(t){var n=this.formattingFn(t);this.el.tagName==="INPUT"?this.el.value=n:this.el.tagName==="text"||this.el.tagName==="tspan"?this.el.textContent=n:this.el.innerHTML=n},i.prototype.ensureNumber=function(t){return typeof t=="number"&&!isNaN(t)},i.prototype.validateValue=function(t){var n=Number(t);return this.ensureNumber(n)?n:(this.error="[CountUp] invalid start or end value: "+t,null)},i.prototype.resetDuration=function(){this.startTime=null,this.duration=1e3*Number(this.options.duration),this.remaining=this.duration},i}();const H={getIndexCount:k.create("/common/index/count","get")},R=[{title:"\u9879\u76EE\u6570",id:"projectNum",num:"123",tip:"\u901A\u8FC7\u4EBA\u6570",tipNum:"911",color:"#FEBB50",iconColor:"#FDC566",icon:"el-icon-histogram"},{title:"Linux\u673A\u5668\u6570",id:"machineNum",num:"123",tip:"\u5728\u573A\u4EBA\u6570",tipNum:"911",color:"#F95959",iconColor:"#F86C6B",icon:"iconfont icon-jinridaiban"},{title:"\u6570\u636E\u5E93\u603B\u6570",id:"dbNum",num:"123",tip:"\u4F7F\u7528\u4E2D",tipNum:"611",color:"#8595F4",iconColor:"#92A1F4",icon:"iconfont icon-AIshiyanshi"},{title:"redis\u603B\u6570",id:"redisNum",num:"123",tip:"\u901A\u8FC7\u4EBA\u6570",tipNum:"911",color:"#1abc9c",iconColor:"#FDC566",icon:"iconfont icon-shenqingkaiban"}],G=[{icon:"iconfont icon-yangan",label:"\u70DF\u611F",value:"2.1%OBS/M",iconColor:"#F72B3F"},{icon:"iconfont icon-wendu",label:"\u6E29\u5EA6",value:"30\u2103",iconColor:"#91BFF8"},{icon:"iconfont icon-shidu",label:"\u6E7F\u5EA6",value:"57%RH",iconColor:"#88D565"},{icon:"iconfont icon-zaosheng",label:"\u566A\u58F0",value:"57DB",iconColor:"#FBD4A0"}],$=[{time1:"\u4ECA\u5929",time2:"12:20:30",title:"\u66F4\u540D",label:"\u6B63\u5F0F\u66F4\u540D\u4E3A vue-next-admin"},{time1:"02-17",time2:"12:20:30",title:"\u9875\u9762",label:"\u5B8C\u6210\u5BF9\u9996\u9875\u7684\u5F00\u53D1"},{time1:"02-14",time2:"12:20:30",title:"\u9875\u9762",label:"\u65B0\u589E\u4E2A\u4EBA\u4E2D\u5FC3"}];var v={name:"Home",setup(){const i=I(),t=S({topCardItemList:R,environmentList:G,activitiesList:$,tableData:{data:[{date:"2016-05-02",name:"1\u53F7\u5B9E\u9A8C\u5BA4",address:"\u70DF\u611F2.1%OBS/M"},{date:"2016-05-04",name:"2\u53F7\u5B9E\u9A8C\u5BA4",address:"\u6E29\u5EA630\u2103"},{date:"2016-05-01",name:"3\u53F7\u5B9E\u9A8C\u5BA4",address:"\u6E7F\u5EA657%RH"}]}}),n=A(()=>T(new Date)),m=async()=>{const r=await H.getIndexCount.request();M(()=>{new c("projectNum",r.projectNum).start(),new c("machineNum",r.machineNum).start(),new c("dbNum",r.dbNum).start(),new c("redisNum",r.redisNum).start()})};j(()=>{m()});const e=A(()=>i.state.userInfos.userInfos);return E({getUserInfos:e,currentTime:n},z(t))}},re=`.home-container[data-v-9bee469e] {
  overflow-x: hidden;
}
.home-container .home-card-item[data-v-9bee469e] {
  width: 100%;
  height: 103px;
  background: gray;
  border-radius: 4px;
  transition: all ease 0.3s;
}
.home-container .home-card-item[data-v-9bee469e]:hover {
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  transition: all ease 0.3s;
}
.home-container .home-card-item-box[data-v-9bee469e] {
  display: flex;
  align-items: center;
  position: relative;
  overflow: hidden;
}
.home-container .home-card-item-box:hover i[data-v-9bee469e] {
  right: 0px !important;
  bottom: 0px !important;
  transition: all ease 0.3s;
}
.home-container .home-card-item-box i[data-v-9bee469e] {
  position: absolute;
  right: -10px;
  bottom: -10px;
  font-size: 70px;
  transform: rotate(-30deg);
  transition: all ease 0.3s;
}
.home-container .home-card-item-box .home-card-item-flex[data-v-9bee469e] {
  padding: 0 20px;
  color: white;
}
.home-container .home-card-item-box .home-card-item-flex .home-card-item-title[data-v-9bee469e],
.home-container .home-card-item-box .home-card-item-flex .home-card-item-tip[data-v-9bee469e] {
  font-size: 13px;
}
.home-container .home-card-item-box .home-card-item-flex .home-card-item-title-num[data-v-9bee469e] {
  font-size: 18px;
}
.home-container .home-card-item-box .home-card-item-flex .home-card-item-tip-num[data-v-9bee469e] {
  font-size: 13px;
}
.home-container .home-card-first[data-v-9bee469e] {
  background: white;
  border: 1px solid #ebeef5;
  display: flex;
  align-items: center;
}
.home-container .home-card-first img[data-v-9bee469e] {
  width: 60px;
  height: 60px;
  border-radius: 100%;
  border: 2px solid var(--color-primary-light-5);
}
.home-container .home-card-first .home-card-first-right[data-v-9bee469e] {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.home-container .home-card-first .home-card-first-right .home-card-first-right-msg[data-v-9bee469e] {
  font-size: 13px;
  color: gray;
}
.home-container .home-monitor[data-v-9bee469e] {
  height: 200px;
}
.home-container .home-monitor .flex-warp-item[data-v-9bee469e] {
  width: 50%;
  height: 100px;
  display: flex;
}
.home-container .home-monitor .flex-warp-item .flex-warp-item-box[data-v-9bee469e] {
  margin: auto;
  height: auto;
  text-align: center;
}
.home-container .home-warning-card[data-v-9bee469e] {
  height: 292px;
}
.home-container .home-warning-card[data-v-9bee469e] .el-card {
  height: 100%;
}
.home-container .home-dynamic[data-v-9bee469e] {
  height: 200px;
}
.home-container .home-dynamic .home-dynamic-item[data-v-9bee469e] {
  display: flex;
  width: 100%;
  height: 60px;
  overflow: hidden;
}
.home-container .home-dynamic .home-dynamic-item:first-of-type .home-dynamic-item-line i[data-v-9bee469e] {
  color: orange !important;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-left[data-v-9bee469e] {
  text-align: right;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-left .home-dynamic-item-left-time2[data-v-9bee469e] {
  font-size: 13px;
  color: gray;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-line[data-v-9bee469e] {
  height: 60px;
  border-right: 2px dashed #dfdfdf;
  margin: 0 20px;
  position: relative;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-line i[data-v-9bee469e] {
  color: var(--color-primary);
  font-size: 12px;
  position: absolute;
  top: 1px;
  left: -6px;
  transform: rotate(46deg);
  background: white;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-right[data-v-9bee469e] {
  flex: 1;
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-right .home-dynamic-item-right-title i[data-v-9bee469e] {
  margin-right: 5px;
  border: 1px solid #dfdfdf;
  width: 20px;
  height: 20px;
  border-radius: 100%;
  padding: 3px 2px 2px;
  text-align: center;
  color: var(--color-primary);
}
.home-container .home-dynamic .home-dynamic-item .home-dynamic-item-right .home-dynamic-item-right-label[data-v-9bee469e] {
  font-size: 13px;
  color: gray;
}`;const p=P();U("data-v-9bee469e");const J={class:"home-container"},K={class:"home-card-item home-card-first"},Q={class:"flex-margin flex"},W={class:"home-card-first-right ml15"},X={class:"flex-margin"},Y={class:"home-card-first-right-title"},Z={class:"home-card-item-flex"},ee={class:"home-card-item-title pb3"};q();const te=p((i,t,n,m,e,r)=>{const o=_("el-col"),l=_("el-row");return g(),x("div",J,[s(l,{gutter:15},{default:p(()=>[s(o,{sm:6,class:"mb15"},{default:p(()=>[s("div",K,[s("div",Q,[s("img",{src:m.getUserInfos.photo},null,8,["src"]),s("div",W,[s("div",X,[s("div",Y,w(`${m.currentTime}, ${m.getUserInfos.username}`),1)])])])])]),_:1}),(g(!0),x(L,null,O(i.topCardItemList,(a,h)=>(g(),x(o,{sm:3,class:"mb15",key:h},{default:p(()=>[s("div",{class:"home-card-item home-card-item-box",style:{background:a.color}},[s("div",Z,[s("div",ee,w(a.title),1),s("div",{class:"home-card-item-title-num pb6",id:a.id},null,8,["id"])]),s("i",{class:a.icon,style:{color:a.iconColor}},null,6)],4)]),_:2},1024))),128))]),_:1})])});v.render=te,v.__scopeId="data-v-9bee469e";export default v;
