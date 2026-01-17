import{y as C,_ as I}from"./js-yaml-A63HP8_m.js";import{d as D,y as $,z as L,u as O,B as h,D as Y,H as x,J as v,w as e,e as s,o as b,b as t,f as a,n as k,ac as M,ad as P,j as o,T as U,m as w,p as H,h as j,_ as z}from"./index-rt1yTeew.js";import{a as J}from"./traffic-nmA_BhEL.js";import{H as K}from"./request-vlI2kaaR.js";const d=p=>(H("data-v-93e60661"),p=p(),j(),p),q={class:"editorBox"},F={class:"bottom-action-footer"},G=d(()=>o("br",null,null,-1)),Q=d(()=>o("br",null,null,-1)),W=d(()=>o("br",null,null,-1)),X=d(()=>o("br",null,null,-1)),Z=d(()=>o("br",null,null,-1)),ee=d(()=>o("br",null,null,-1)),te=D({__name:"addByYAMLView",setup(p){const c=$(L.TAB_LAYOUT_STATE),A=O(),B=h(!1),r=h(!1),R=h(8),i=h(`conditions:
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
scope: service`);Y(()=>{if(x.isNil(c.conditionRule))i.value="";else{const l=c.conditionRule;i.value=C.dump(l)}});const E=l=>{c.conditionRule=C.load(i.value)},N=async()=>{const l=C.load(i.value),{configVersion:_,scope:g,key:u,runtime:S,force:T,conditions:y}=l;let n="";if(u=="application")n=`${u}.condition-router`;else if(x.isNil(c.addConditionRuleSate)){w.error("请先填写版本和分组字段");return}else{const{version:m,group:f}=c.addConditionRuleSate;if(m==""||f==""){w.error("请先填写版本和分组字段");return}n=`${u}:${m}:${f}.condition-router`}l.configVersion="v3.0",(await J(n,l)).code===K.SUCCESS&&A.push("/traffic/routingRule")};return(l,_)=>{const g=s("a-button"),u=s("a-flex"),S=s("a-space"),T=s("a-affix"),y=s("a-col"),n=s("a-descriptions-item"),V=s("a-descriptions"),m=s("a-card");return b(),v(m,null,{default:e(()=>[t(u,{style:{width:"100%"}},{default:e(()=>[t(y,{span:r.value?24-R.value:24,class:"left"},{default:e(()=>[t(u,{vertical:"",align:"end"},{default:e(()=>[t(g,{type:"text",style:{color:"#0a90d5"},onClick:_[0]||(_[0]=f=>r.value=!r.value)},{default:e(()=>[a(" 字段说明 "),r.value?(b(),v(k(P),{key:1})):(b(),v(k(M),{key:0}))]),_:1}),o("div",q,[t(I,{onChange:E,modelValue:i.value,"onUpdate:modelValue":_[1]||(_[1]=f=>i.value=f),theme:"vs-dark",height:500,language:"yaml",readonly:B.value},null,8,["modelValue","readonly"])])]),_:1}),t(T,{"offset-bottom":10},{default:e(()=>[o("div",F,[t(S,{align:"center",size:"large"},{default:e(()=>[t(g,{type:"primary",onClick:N},{default:e(()=>[a(" 确认 ")]),_:1}),t(g,null,{default:e(()=>[a(" 取消 ")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),t(y,{span:r.value?R.value:0,class:"right"},{default:e(()=>[r.value?(b(),v(m,{key:0,class:"sliderBox"},{default:e(()=>[o("div",null,[t(V,{title:"字段说明",column:1},{default:e(()=>[t(n,{label:"key"},{default:e(()=>[a(" 作用对象"),G,a(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),t(n,{label:"scope"},{default:e(()=>[a(" 规则粒度"),Q,a(" 可能的值：application, service ")]),_:1}),t(n,{label:"force"},{default:e(()=>[a(" 容错保护"),W,a(" 可能的值：true, false"),X,a(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),t(n,{label:"runtime"},{default:e(()=>[a(" 运行时生效"),Z,a(" 可能的值：true, false"),ee,a(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):U("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),le=z(te,[["__scopeId","data-v-93e60661"]]);export{le as default};
