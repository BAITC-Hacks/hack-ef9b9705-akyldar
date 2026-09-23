const base = { title:'',initial_description:'',context:'',need:'',users:'',data:'',constraints:'',expected_result:'',success_criteria:'',contact:'',interaction_format:'',topic:'',rating:0,readiness_level:'draft',confirmed:false,published:false,created_at:'',updated_at:'' };
export const initialTasks = [
  { ...base,id:'sample-1',title:'AI inventory assistant',context:'A regional retailer manages inventory across three warehouses using spreadsheets.',need:'Help staff predict stock shortages and reorder products on time.',users:'Warehouse managers and purchasing team',data:'Two years of sales records, SKU list and daily stock exports.',constraints:'A browser-based prototype; use anonymized data only.',expected_result:'A dashboard with stock alerts and a simple demand forecast.',success_criteria:'Reduce stockouts by 20% during a four-week pilot.',contact:'innovation@northstar.kz',interaction_format:'Weekly online check-ins',topic:'AI & Data',status:'published',scoreOverride:95 },
  { ...base,id:'sample-2',title:'Smarter delivery routes',context:'Our couriers lose time on inefficient routes.',need:'Plan daily delivery routes for city couriers.',users:'Dispatchers and couriers',data:'Anonymized delivery coordinates and delivery windows.',constraints:'Prototype for one city.',expected_result:'A map with optimized daily routes.',success_criteria:'Cut total travel time by 15%.',contact:'hello@flow.kz',topic:'Logistics',status:'published',scoreOverride:82 },
  { ...base,id:'sample-3',title:'Customer feedback insights',context:'Our team reads hundreds of customer comments each week.',need:'Find recurring pain points in feedback.',users:'Customer support managers',constraints:'Do not expose personal customer data.',expected_result:'A summary dashboard of top themes.',contact:'team@voice.kz',topic:'AI & Data',status:'published',scoreOverride:67 },
  { ...base,id:'sample-4',title:'Campus café queue',context:'Students wait too long at lunch.',need:'Explore ways to reduce queues.',users:'Café staff and students',topic:'Food & Retail',status:'published',scoreOverride:48 },
  { ...base,id:'sample-5',title:'A better website',need:'We want a new website.',topic:'Technology',status:'published',scoreOverride:25 }
];
export const fields = [
  ['title','Task title','text'],['context','Context','textarea'],['need','Business need','textarea'],['users','Who will use it?','text'],['data','Available data & materials','textarea'],['constraints','Constraints','textarea'],['expected_result','Expected result','textarea'],['success_criteria','Success criteria','textarea'],['contact','Business contact','text'],['interaction_format','Interaction format','text'],['topic','Topic','text']
];
export const breakdown = [
  ['Context & need',20,'context_need'],['Available data',20,'data'],['Expected result',15,'expected_result'],['Success criteria',15,'success_criteria'],['Constraints',10,'constraints'],['Users',10,'users'],['Business connection',10,'business_connection']
];
export function rating(task){
  if(task.scoreOverride != null) return task.scoreOverride;
  if(task.rating != null && Number.isFinite(Number(task.rating))) return Number(task.rating);
  return calculateMockRating(task).score;
}
export function readiness(score){return score<40?'draft':score<70?'working':score<90?'ready':'priority'}
export function readinessLabel(level){return String(level||'draft').replace(/^./,letter=>letter.toUpperCase())}
export const questions=['Who will use the solution day to day?','What data or materials can you share with the student team?','How will you know that the solution succeeded?'];
export const fallbackQuestions=questions;
export function calculateMockRating(task={}){
  const has=key=>Boolean(String(task[key]||'').trim());
  const values={context_need:has('context')&&has('need')?20:0,data:has('data')?20:0,expected_result:has('expected_result')?15:0,success_criteria:has('success_criteria')?15:0,constraints:has('constraints')?10:0,users:has('users')?10:0,business_connection:(has('contact')?5:0)+(has('interaction_format')?5:0)};
  const missing=['context','need','data','expected_result','success_criteria','constraints','users','contact','interaction_format'].filter(hasKey=>!has(hasKey));
  const score=Object.values(values).reduce((total,value)=>total+value,0);
  return {score,level:readiness(score),breakdown:values,missing};
}
export function mockCard(description,answers){return {title:'',context:String(description||'').trim(),need:'',users:answers?.[0]?.answer?.trim()||'',data:answers?.[1]?.answer?.trim()||'',constraints:'',expected_result:'',success_criteria:answers?.[2]?.answer?.trim()||'',contact:'',interaction_format:'',topic:''};}
export function mockTask(description,topic=''){
  const now=new Date().toISOString();
  const task={...base,id:crypto.randomUUID(),initial_description:String(description||'').trim(),topic:String(topic||'').trim(),created_at:now,updated_at:now};
  const score=calculateMockRating(task);
  return {...task,...score,rating:score.score,readiness_level:score.level};
}
export function freshTask(description,answers){const task={...mockTask(description),...mockCard(description,answers)};const score=calculateMockRating(task);return {...task,...score,rating:score.score,readiness_level:score.level};}
