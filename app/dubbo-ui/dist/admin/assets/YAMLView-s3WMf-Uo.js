import{y as D,_ as b}from"./js-yaml-eElisXzH.js";import{e as B}from"./traffic-dHGZ6qwp.js";import{d as C,a as L,B as u,D as R,J as r,w as e,e as c,o as s,b as a,f as p,t as I,n as x,aa as M,ab as N,j as f,c as S,M as A,L as T,T as O,p as Y,h as $,_ as j}from"./index-3zDsduUv.js";import"./request-3an337VF.js";const h=n=>(Y("data-v-4f1417af"),n=n(),$(),n),q={class:"editorBox"},E=h(()=>f("p",null,"修改时间: 2024/3/20 15:20:31",-1)),F=h(()=>f("p",null,"版本号: xo842xqpx834",-1)),J=C({__name:"YAMLView",setup(n){const k=L(),V=u(!0),t=u(!1),m=u(8),y=u(`configVersion: v3.0
force: true
enabled: true
key: shop-detail
tags:
  - name: gray
    match:
      - key: env
        value:
          exact: gray`),w=async()=>{var l;const o=await B((l=k.params)==null?void 0:l.ruleName);o.code===200&&(y.value=D.dump(o==null?void 0:o.data))};return R(()=>{w()}),(o,l)=>{const d=c("a-button"),_=c("a-flex"),v=c("a-col"),i=c("a-card");return s(),r(i,null,{default:e(()=>[a(_,{style:{width:"100%"}},{default:e(()=>[a(v,{span:t.value?24-m.value:24,class:"left"},{default:e(()=>[a(_,{vertical:"",align:"end"},{default:e(()=>[a(d,{type:"text",style:{color:"#0a90d5"},onClick:l[0]||(l[0]=g=>t.value=!t.value)},{default:e(()=>[p(I(o.$t("flowControlDomain.versionRecords"))+" ",1),t.value?(s(),r(x(N),{key:1})):(s(),r(x(M),{key:0}))]),_:1}),f("div",q,[a(b,{modelValue:y.value,theme:"vs-dark",height:500,language:"yaml",readonly:V.value},null,8,["modelValue","readonly"])])]),_:1})]),_:1},8,["span"]),a(v,{span:t.value?m.value:0,class:"right"},{default:e(()=>[t.value?(s(),r(i,{key:0,class:"sliderBox"},{default:e(()=>[(s(),S(T,null,A(2,g=>a(i,{key:g},{default:e(()=>[E,F,a(_,{justify:"flex-end"},{default:e(()=>[a(d,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[p("查看")]),_:1}),a(d,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[p("回滚")]),_:1})]),_:1})]),_:2},1024)),64))]),_:1})):O("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),K=j(J,[["__scopeId","data-v-4f1417af"]]);export{K as default};
