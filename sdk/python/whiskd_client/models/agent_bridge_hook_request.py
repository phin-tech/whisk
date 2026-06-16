from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_bridge_hook_decision import AgentBridgeHookDecision
    from ..models.agent_bridge_hook_request_raw_payload import (
        AgentBridgeHookRequestRawPayload,
    )
    from ..models.agent_bridge_hook_request_tool_input import (
        AgentBridgeHookRequestToolInput,
    )


T = TypeVar("T", bound="AgentBridgeHookRequest")


@_attrs_define
class AgentBridgeHookRequest:
    """
    Attributes:
        event_name (str):
        provider (str):
        token (str):
        action (str | Unset):
        decision (AgentBridgeHookDecision | Unset):
        elicitation_id (str | Unset):
        message (str | Unset):
        notification_type (str | Unset):
        pty_id (str | Unset):
        raw_payload (AgentBridgeHookRequestRawPayload | Unset):
        session_id (str | Unset):
        tool_input (AgentBridgeHookRequestToolInput | Unset):
        tool_name (str | Unset):
        tool_output (str | Unset):
    """

    event_name: str
    provider: str
    token: str
    action: str | Unset = UNSET
    decision: AgentBridgeHookDecision | Unset = UNSET
    elicitation_id: str | Unset = UNSET
    message: str | Unset = UNSET
    notification_type: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    raw_payload: AgentBridgeHookRequestRawPayload | Unset = UNSET
    session_id: str | Unset = UNSET
    tool_input: AgentBridgeHookRequestToolInput | Unset = UNSET
    tool_name: str | Unset = UNSET
    tool_output: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        event_name = self.event_name

        provider = self.provider

        token = self.token

        action = self.action

        decision: dict[str, Any] | Unset = UNSET
        if not isinstance(self.decision, Unset):
            decision = self.decision.to_dict()

        elicitation_id = self.elicitation_id

        message = self.message

        notification_type = self.notification_type

        pty_id = self.pty_id

        raw_payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.raw_payload, Unset):
            raw_payload = self.raw_payload.to_dict()

        session_id = self.session_id

        tool_input: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tool_input, Unset):
            tool_input = self.tool_input.to_dict()

        tool_name = self.tool_name

        tool_output = self.tool_output

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "eventName": event_name,
                "provider": provider,
                "token": token,
            }
        )
        if action is not UNSET:
            field_dict["action"] = action
        if decision is not UNSET:
            field_dict["decision"] = decision
        if elicitation_id is not UNSET:
            field_dict["elicitationId"] = elicitation_id
        if message is not UNSET:
            field_dict["message"] = message
        if notification_type is not UNSET:
            field_dict["notificationType"] = notification_type
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if raw_payload is not UNSET:
            field_dict["rawPayload"] = raw_payload
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if tool_input is not UNSET:
            field_dict["toolInput"] = tool_input
        if tool_name is not UNSET:
            field_dict["toolName"] = tool_name
        if tool_output is not UNSET:
            field_dict["toolOutput"] = tool_output

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_bridge_hook_decision import AgentBridgeHookDecision
        from ..models.agent_bridge_hook_request_raw_payload import (
            AgentBridgeHookRequestRawPayload,
        )
        from ..models.agent_bridge_hook_request_tool_input import (
            AgentBridgeHookRequestToolInput,
        )

        d = dict(src_dict)
        event_name = d.pop("eventName")

        provider = d.pop("provider")

        token = d.pop("token")

        action = d.pop("action", UNSET)

        _decision = d.pop("decision", UNSET)
        decision: AgentBridgeHookDecision | Unset
        if isinstance(_decision, Unset):
            decision = UNSET
        else:
            decision = AgentBridgeHookDecision.from_dict(_decision)

        elicitation_id = d.pop("elicitationId", UNSET)

        message = d.pop("message", UNSET)

        notification_type = d.pop("notificationType", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        _raw_payload = d.pop("rawPayload", UNSET)
        raw_payload: AgentBridgeHookRequestRawPayload | Unset
        if isinstance(_raw_payload, Unset):
            raw_payload = UNSET
        else:
            raw_payload = AgentBridgeHookRequestRawPayload.from_dict(_raw_payload)

        session_id = d.pop("sessionId", UNSET)

        _tool_input = d.pop("toolInput", UNSET)
        tool_input: AgentBridgeHookRequestToolInput | Unset
        if isinstance(_tool_input, Unset):
            tool_input = UNSET
        else:
            tool_input = AgentBridgeHookRequestToolInput.from_dict(_tool_input)

        tool_name = d.pop("toolName", UNSET)

        tool_output = d.pop("toolOutput", UNSET)

        agent_bridge_hook_request = cls(
            event_name=event_name,
            provider=provider,
            token=token,
            action=action,
            decision=decision,
            elicitation_id=elicitation_id,
            message=message,
            notification_type=notification_type,
            pty_id=pty_id,
            raw_payload=raw_payload,
            session_id=session_id,
            tool_input=tool_input,
            tool_name=tool_name,
            tool_output=tool_output,
        )

        agent_bridge_hook_request.additional_properties = d
        return agent_bridge_hook_request

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
