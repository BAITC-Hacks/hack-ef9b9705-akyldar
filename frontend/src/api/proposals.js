import {request} from './client.js';

export function createProposal(taskId,payload){return request(`/api/tasks/${taskId}/proposals`,{method:'POST',body:JSON.stringify(payload)})}
export function getProposals(taskId){return request(`/api/tasks/${taskId}/proposals`)}
export function updateProposalStatus(id,status){return request(`/api/proposals/${id}/status`,{method:'PATCH',body:JSON.stringify({status})})}
