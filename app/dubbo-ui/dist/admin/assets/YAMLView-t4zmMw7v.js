import{y as b,_ as w}from"./js-yaml-jCuc0YZg.js";import{e as C}from"./traffic-CzL2emsx.js";import{d as D,a as T,B as d,D as B,J as r,w as e,e as u,o as l,b as a,f,t as L,n as x,aa as R,ab as A,j as p,c as I,M,L as N,T as O,p as Y,h as $,_ as j}from"./index-HmDwMLmp.js";import{H as E}from"./request-1uc-Zik6.js";const h=n=>(Y("data-v-f2b8fdfe"),n=n(),$(),n),H={class:"editorBox"},P=h(()=>p("p",null,"修改时间: 2024/3/20 15:20:31",-1)),U=h(()=>p("p",null,"版本号: xo842xqpx834",-1)),q=D({__name:"YAMLView",setup(n){const k=T(),S=d(!0),t=d(!1),m=d(8),y=d(`configVersion: v3.0
force: true
enabled: true
key: shop-detail
tags:
  - name: gray
    match:
      - key: env
        value:
          exact: gray`),V=async()=>{var s;const o=await C((s=k.params)==null?void 0:s.ruleName);o.code===E.SUCCESS&&(y.value=b.dump(o==null?void 0:o.data))};return B(()=>{V()}),(o,s)=>{const c=u("a-button"),_=u("a-flex"),v=u("a-col"),i=u("a-card");return l(),r(i,null,{default:e(()=>[a(_,{style:{width:"100%"}},{default:e(()=>[a(v,{span:t.value?24-m.value:24,class:"left"},{default:e(()=>[a(_,{vertical:"",align:"end"},{default:e(()=>[a(c,{type:"text",style:{color:"#0a90d5"},onClick:s[0]||(s[0]=g=>t.value=!t.value)},{default:e(()=>[f(L(o.$t("flowControlDomain.versionRecords"))+" ",1),t.value?(l(),r(x(A),{key:1})):(l(),r(x(R),{key:0}))]),_:1}),p("div",H,[a(w,{modelValue:y.value,theme:"vs-dark",height:500,language:"yaml",readonly:S.value},null,8,["modelValue","readonly"])])]),_:1})]),_:1},8,["span"]),a(v,{span:t.value?m.value:0,class:"right"},{default:e(()=>[t.value?(l(),r(i,{key:0,class:"sliderBox"},{default:e(()=>[(l(),I(N,null,M(2,g=>a(i,{key:g},{default:e(()=>[P,U,a(_,{justify:"flex-end"},{default:e(()=>[a(c,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[f("查看")]),_:1}),a(c,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[f("回滚")]),_:1})]),_:1})]),_:2},1024)),64))]),_:1})):O("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),K=j(q,[["__scopeId","data-v-f2b8fdfe"]]);export{K as default};
