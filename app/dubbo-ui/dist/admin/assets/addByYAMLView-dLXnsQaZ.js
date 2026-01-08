import{y as C,_ as A}from"./js-yaml-w13nwLeX.js";import{d as D,y as $,z as O,u as P,B as h,D as Y,H as k,J as v,w as e,e as s,o as b,b as t,f as a,n as w,aa as L,ab as M,j as o,T as H,m as T,p as J,h as U,_ as j}from"./index-pQAIy9zr.js";import{a as z}from"./traffic-RdeKrzRA.js";import{H as K}from"./request-FtkW8_Rp.js";const c=p=>(J("data-v-06ec3f5e"),p=p(),U(),p),q={class:"editorBox"},F={class:"bottom-action-footer"},G=c(()=>o("br",null,null,-1)),Q=c(()=>o("br",null,null,-1)),W=c(()=>o("br",null,null,-1)),X=c(()=>o("br",null,null,-1)),Z=c(()=>o("br",null,null,-1)),ee=c(()=>o("br",null,null,-1)),te=D({__name:"addByYAMLView",setup(p){const d=$(O.PROVIDE_INJECT_KEY),E=P(),B=h(!1),r=h(!1),R=h(8),i=h(`conditions:
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
scope: service`);Y(()=>{if(k.isNil(d.conditionRule))i.value="";else{const l=d.conditionRule;i.value=C.dump(l)}});const I=l=>{d.conditionRule=C.load(i.value)},N=async()=>{const l=C.load(i.value),{configVersion:_,scope:g,key:u,runtime:V,force:x,conditions:y}=l;let n="";if(u=="application")n=`${u}.condition-router`;else if(k.isNil(d.addConditionRuleSate)){T.error("请先填写版本和分组字段");return}else{const{version:m,group:f}=d.addConditionRuleSate;if(m==""||f==""){T.error("请先填写版本和分组字段");return}n=`${u}:${m}:${f}.condition-router`}l.configVersion="v3.0",(await z(n,l)).code===K.SUCCESS&&E.push("/traffic/routingRule")};return(l,_)=>{const g=s("a-button"),u=s("a-flex"),V=s("a-space"),x=s("a-affix"),y=s("a-col"),n=s("a-descriptions-item"),S=s("a-descriptions"),m=s("a-card");return b(),v(m,null,{default:e(()=>[t(u,{style:{width:"100%"}},{default:e(()=>[t(y,{span:r.value?24-R.value:24,class:"left"},{default:e(()=>[t(u,{vertical:"",align:"end"},{default:e(()=>[t(g,{type:"text",style:{color:"#0a90d5"},onClick:_[0]||(_[0]=f=>r.value=!r.value)},{default:e(()=>[a(" 字段说明 "),r.value?(b(),v(w(M),{key:1})):(b(),v(w(L),{key:0}))]),_:1}),o("div",q,[t(A,{onChange:I,modelValue:i.value,"onUpdate:modelValue":_[1]||(_[1]=f=>i.value=f),theme:"vs-dark",height:500,language:"yaml",readonly:B.value},null,8,["modelValue","readonly"])])]),_:1}),t(x,{"offset-bottom":10},{default:e(()=>[o("div",F,[t(V,{align:"center",size:"large"},{default:e(()=>[t(g,{type:"primary",onClick:N},{default:e(()=>[a(" 确认 ")]),_:1}),t(g,null,{default:e(()=>[a(" 取消 ")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),t(y,{span:r.value?R.value:0,class:"right"},{default:e(()=>[r.value?(b(),v(m,{key:0,class:"sliderBox"},{default:e(()=>[o("div",null,[t(S,{title:"字段说明",column:1},{default:e(()=>[t(n,{label:"key"},{default:e(()=>[a(" 作用对象"),G,a(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),t(n,{label:"scope"},{default:e(()=>[a(" 规则粒度"),Q,a(" 可能的值：application, service ")]),_:1}),t(n,{label:"force"},{default:e(()=>[a(" 容错保护"),W,a(" 可能的值：true, false"),X,a(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),t(n,{label:"runtime"},{default:e(()=>[a(" 运行时生效"),Z,a(" 可能的值：true, false"),ee,a(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):H("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),le=j(te,[["__scopeId","data-v-06ec3f5e"]]);export{le as default};
