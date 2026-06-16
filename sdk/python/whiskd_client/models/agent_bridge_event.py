from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_bridge_event_raw import AgentBridgeEventRaw


T = TypeVar("T", bound="AgentBridgeEvent")


@_attrs_define
class AgentBridgeEvent:
    """
    Attributes:
        created_at (datetime.datetime):
        event_name (str):
        id (str):
        provider (str):
        status (str):
        action (str | Unset):
        bridge_id (str | Unset):
        elicitation_id (str | Unset):
        message (str | Unset):
        notification_type (str | Unset):
        pty_id (str | Unset):
        raw (AgentBridgeEventRaw | Unset):
        result (str | Unset):
        session_id (str | Unset):
        tool_name (str | Unset):
    """

    created_at: datetime.datetime
    event_name: str
    id: str
    provider: str
    status: str
    action: str | Unset = UNSET
    bridge_id: str | Unset = UNSET
    elicitation_id: str | Unset = UNSET
    message: str | Unset = UNSET
    notification_type: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    raw: AgentBridgeEventRaw | Unset = UNSET
    result: str | Unset = UNSET
    session_id: str | Unset = UNSET
    tool_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        event_name = self.event_name

        id = self.id

        provider = self.provider

        status = self.status

        action = self.action

        bridge_id = self.bridge_id

        elicitation_id = self.elicitation_id

        message = self.message

        notification_type = self.notification_type

        pty_id = self.pty_id

        raw: dict[str, Any] | Unset = UNSET
        if not isinstance(self.raw, Unset):
            raw = self.raw.to_dict()

        result = self.result

        session_id = self.session_id

        tool_name = self.tool_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "eventName": event_name,
                "id": id,
                "provider": provider,
                "status": status,
            }
        )
        if action is not UNSET:
            field_dict["action"] = action
        if bridge_id is not UNSET:
            field_dict["bridgeId"] = bridge_id
        if elicitation_id is not UNSET:
            field_dict["elicitationId"] = elicitation_id
        if message is not UNSET:
            field_dict["message"] = message
        if notification_type is not UNSET:
            field_dict["notificationType"] = notification_type
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if raw is not UNSET:
            field_dict["raw"] = raw
        if result is not UNSET:
            field_dict["result"] = result
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if tool_name is not UNSET:
            field_dict["toolName"] = tool_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_bridge_event_raw import AgentBridgeEventRaw

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        event_name = d.pop("eventName")

        id = d.pop("id")

        provider = d.pop("provider")

        status = d.pop("status")

        action = d.pop("action", UNSET)

        bridge_id = d.pop("bridgeId", UNSET)

        elicitation_id = d.pop("elicitationId", UNSET)

        message = d.pop("message", UNSET)

        notification_type = d.pop("notificationType", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        _raw = d.pop("raw", UNSET)
        raw: AgentBridgeEventRaw | Unset
        if isinstance(_raw, Unset):
            raw = UNSET
        else:
            raw = AgentBridgeEventRaw.from_dict(_raw)

        result = d.pop("result", UNSET)

        session_id = d.pop("sessionId", UNSET)

        tool_name = d.pop("toolName", UNSET)

        agent_bridge_event = cls(
            created_at=created_at,
            event_name=event_name,
            id=id,
            provider=provider,
            status=status,
            action=action,
            bridge_id=bridge_id,
            elicitation_id=elicitation_id,
            message=message,
            notification_type=notification_type,
            pty_id=pty_id,
            raw=raw,
            result=result,
            session_id=session_id,
            tool_name=tool_name,
        )

        agent_bridge_event.additional_properties = d
        return agent_bridge_event

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
