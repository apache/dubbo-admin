import{y as k,_ as w}from"./js-yaml-w13nwLeX.js";import{e as T}from"./traffic-RdeKrzRA.js";import{d as B,a as b,B as s,D as C,J as y,w as e,e as n,o as u,b as a,j as _,c as A,M as I,f as v,L,T as M,p as N,h as D,_ as R}from"./index-pQAIy9zr.js";import{H as Y}from"./request-FtkW8_Rp.js";const x=o=>(N("data-v-cdabac6c"),o=o(),D(),o),j={class:"editorBox"},E=x(()=>_("p",null,"修改时间: 2024/3/20 15:20:31",-1)),H=x(()=>_("p",null,"版本号: xo842xqpx834",-1)),P=B({__name:"YAMLView",setup(o){const h=b(),g=s(!0),l=s(!1),p=s(8),i=s(`configVersion: v3.0
force: true
enabled: true
key: shop-detail
tags:
  - name: gray
    match:
      - key: env
        value:
          exact: gray`),V=async()=>{var c;const t=await T((c=h.params)==null?void 0:c.ruleName);t.code===Y.SUCCESS&&(i.value=k.dump(t==null?void 0:t.data))};return C(()=>{V()}),(t,c)=>{const d=n("a-flex"),m=n("a-col"),f=n("a-button"),r=n("a-card");return u(),y(r,null,{default:e(()=>[a(d,{style:{width:"100%"}},{default:e(()=>[a(m,{span:l.value?24-p.value:24,class:"left"},{default:e(()=>[a(d,{vertical:"",align:"end"},{default:e(()=>[_("div",j,[a(w,{modelValue:i.value,theme:"vs-dark",height:500,language:"yaml",readonly:g.value},null,8,["modelValue","readonly"])])]),_:1})]),_:1},8,["span"]),a(m,{span:l.value?p.value:0,class:"right"},{default:e(()=>[l.value?(u(),y(r,{key:0,class:"sliderBox"},{default:e(()=>[(u(),A(L,null,I(2,S=>a(r,{key:S},{default:e(()=>[E,H,a(d,{justify:"flex-end"},{default:e(()=>[a(f,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[v("查看")]),_:1}),a(f,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[v("回滚")]),_:1})]),_:1})]),_:2},1024)),64))]),_:1})):M("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),O=R(P,[["__scopeId","data-v-cdabac6c"]]);export{O as default};
