import{y as f,_ as A}from"./js-yaml-8Gkz3BRW.js";import{g as S,u as T}from"./traffic-C2a-KjHH.js";import{d as O,x as P,y as Y,a as L,z as g,W as M,F as j,G as h,w as e,e as u,o as v,b as a,f as s,n as I,a9 as z,aa as J,j as n,Q as K,m as $,p as F,h as G,_ as Q}from"./index-hmLAZQYT.js";import"./request-8jI_GZey.js";const d=_=>(F("data-v-f0b8727f"),_=_(),G(),_),U={class:"editorBox"},W={class:"bottom-action-footer"},q=d(()=>n("br",null,null,-1)),H=d(()=>n("br",null,null,-1)),X=d(()=>n("br",null,null,-1)),Z=d(()=>n("br",null,null,-1)),ee=d(()=>n("br",null,null,-1)),te=d(()=>n("br",null,null,-1)),ae=O({__name:"updateByYAMLView",setup(_){const b=P(Y.PROVIDE_INJECT_KEY),y=L(),E=g(!1),i=g(!1),x=g(8),r=g(`conditions:
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[method]]]=string &
        arguments[arguments[arguments[arguments[string]]]]!=string
    to:
      - match: string!=string
        weight: 0
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[string]]]!=string
    to:
      - match: string!=lggbond
        weight: 0
      - match: ss!=ss
        weight: 0
configVersion: v3.1
enabled: true
force: false
key: org.apache.dubbo.samples.CommentService
runtime: true
scope: service`);M(()=>{if(j.isNil(b.conditionRule))r.value="",V();else{const t=b.conditionRule;r.value=f.dump(t)}});const N=t=>{b.conditionRule=f.load(r.value)};async function V(){var l,o,m;let t=await S((l=y.params)==null?void 0:l.ruleName);if((t==null?void 0:t.code)===200){const c=(o=y.params)==null?void 0:o.ruleName;if(c&&t.data.scope==="service"){const R=c==null?void 0:c.split(":");t.data.group=(m=R[2])==null?void 0:m.split(".")[0]}r.value=f.dump(t==null?void 0:t.data)}}const B=async()=>{var o;const t=f.load(r.value);t.configVersion="v3.0",(await T((o=y.params)==null?void 0:o.ruleName,t)).code===200&&(await V(),$.success("修改成功"))};return(t,l)=>{const o=u("a-button"),m=u("a-flex"),c=u("a-space"),R=u("a-affix"),w=u("a-col"),p=u("a-descriptions-item"),D=u("a-descriptions"),C=u("a-card");return v(),h(C,null,{default:e(()=>[a(m,{style:{width:"100%"}},{default:e(()=>[a(w,{span:i.value?24-x.value:24,class:"left"},{default:e(()=>[a(m,{vertical:"",align:"end"},{default:e(()=>[a(o,{type:"text",style:{color:"#0a90d5"},onClick:l[0]||(l[0]=k=>i.value=!i.value)},{default:e(()=>[s(" 字段说明 "),i.value?(v(),h(I(J),{key:1})):(v(),h(I(z),{key:0}))]),_:1}),n("div",U,[a(A,{modelValue:r.value,"onUpdate:modelValue":l[1]||(l[1]=k=>r.value=k),theme:"vs-dark",height:500,language:"yaml",readonly:E.value,onChange:N},null,8,["modelValue","readonly"])])]),_:1}),a(R,{"offset-bottom":10},{default:e(()=>[n("div",W,[a(c,{align:"center",size:"large"},{default:e(()=>[a(o,{type:"primary",onClick:B},{default:e(()=>[s(" 确认")]),_:1}),a(o,null,{default:e(()=>[s(" 取消")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),a(w,{span:i.value?x.value:0,class:"right"},{default:e(()=>[i.value?(v(),h(C,{key:0,class:"sliderBox"},{default:e(()=>[n("div",null,[a(D,{title:"字段说明",column:1},{default:e(()=>[a(p,{label:"key"},{default:e(()=>[s(" 作用对象"),q,s(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),a(p,{label:"scope"},{default:e(()=>[s(" 规则粒度"),H,s(" 可能的值：application, service ")]),_:1}),a(p,{label:"force"},{default:e(()=>[s(" 容错保护"),X,s(" 可能的值：true, false"),Z,s(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),a(p,{label:"runtime"},{default:e(()=>[s(" 运行时生效"),ee,s(" 可能的值：true, false"),te,s(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):K("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),ue=Q(ae,[["__scopeId","data-v-f0b8727f"]]);export{ue as default};
