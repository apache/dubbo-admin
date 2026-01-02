import{y as f,_ as N}from"./js-yaml-jCuc0YZg.js";import{g as A,u as P}from"./traffic-CzL2emsx.js";import{d as O,y as Y,z as L,a as M,B as g,D as U,H,J as h,w as e,e as u,o as v,b as a,f as s,n as E,aa as J,ab as j,j as n,T as z,m as K,p as $,h as q,_ as F}from"./index-HmDwMLmp.js";import{H as T}from"./request-1uc-Zik6.js";const d=_=>($("data-v-0e6a5ad3"),_=_(),q(),_),G={class:"editorBox"},Q={class:"bottom-action-footer"},W=d(()=>n("br",null,null,-1)),X=d(()=>n("br",null,null,-1)),Z=d(()=>n("br",null,null,-1)),ee=d(()=>n("br",null,null,-1)),te=d(()=>n("br",null,null,-1)),ae=d(()=>n("br",null,null,-1)),se=O({__name:"updateByYAMLView",setup(_){const b=Y(L.PROVIDE_INJECT_KEY),y=M(),k=g(!1),i=g(!1),R=g(8),r=g(`conditions:
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
scope: service`);U(()=>{if(H.isNil(b.conditionRule))r.value="",S();else{const t=b.conditionRule;r.value=f.dump(t)}});const I=t=>{b.conditionRule=f.load(r.value)};async function S(){var l,o,m;let t=await A((l=y.params)==null?void 0:l.ruleName);if((t==null?void 0:t.code)===T.SUCCESS){const c=(o=y.params)==null?void 0:o.ruleName;if(c&&t.data.scope==="service"){const C=c==null?void 0:c.split(":");t.data.group=(m=C[2])==null?void 0:m.split(".")[0]}r.value=f.dump(t==null?void 0:t.data)}}const B=async()=>{var o;const t=f.load(r.value);t.configVersion="v3.0",(await P((o=y.params)==null?void 0:o.ruleName,t)).code===T.SUCCESS&&(await S(),K.success("修改成功"))};return(t,l)=>{const o=u("a-button"),m=u("a-flex"),c=u("a-space"),C=u("a-affix"),V=u("a-col"),p=u("a-descriptions-item"),D=u("a-descriptions"),w=u("a-card");return v(),h(w,null,{default:e(()=>[a(m,{style:{width:"100%"}},{default:e(()=>[a(V,{span:i.value?24-R.value:24,class:"left"},{default:e(()=>[a(m,{vertical:"",align:"end"},{default:e(()=>[a(o,{type:"text",style:{color:"#0a90d5"},onClick:l[0]||(l[0]=x=>i.value=!i.value)},{default:e(()=>[s(" 字段说明 "),i.value?(v(),h(E(j),{key:1})):(v(),h(E(J),{key:0}))]),_:1}),n("div",G,[a(N,{modelValue:r.value,"onUpdate:modelValue":l[1]||(l[1]=x=>r.value=x),theme:"vs-dark",height:500,language:"yaml",readonly:k.value,onChange:I},null,8,["modelValue","readonly"])])]),_:1}),a(C,{"offset-bottom":10},{default:e(()=>[n("div",Q,[a(c,{align:"center",size:"large"},{default:e(()=>[a(o,{type:"primary",onClick:B},{default:e(()=>[s(" 确认")]),_:1}),a(o,null,{default:e(()=>[s(" 取消")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),a(V,{span:i.value?R.value:0,class:"right"},{default:e(()=>[i.value?(v(),h(w,{key:0,class:"sliderBox"},{default:e(()=>[n("div",null,[a(D,{title:"字段说明",column:1},{default:e(()=>[a(p,{label:"key"},{default:e(()=>[s(" 作用对象"),W,s(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),a(p,{label:"scope"},{default:e(()=>[s(" 规则粒度"),X,s(" 可能的值：application, service ")]),_:1}),a(p,{label:"force"},{default:e(()=>[s(" 容错保护"),Z,s(" 可能的值：true, false"),ee,s(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),a(p,{label:"runtime"},{default:e(()=>[s(" 运行时生效"),te,s(" 可能的值：true, false"),ae,s(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):z("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),re=F(se,[["__scopeId","data-v-0e6a5ad3"]]);export{re as default};
