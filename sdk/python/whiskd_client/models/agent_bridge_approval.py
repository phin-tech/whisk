from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_bridge_approval_tool_input import AgentBridgeApprovalToolInput
    from ..models.agent_bridge_hook_decision import AgentBridgeHookDecision


T = TypeVar("T", bound="AgentBridgeApproval")


@_attrs_define
class AgentBridgeApproval:
    """
    Attributes:
        bridge_id (str):
        created_at (datetime.datetime):
        event_name (str):
        id (str):
        provider (str):
        status (str):
        tool_name (str):
        decision (AgentBridgeHookDecision | Unset):
        pty_id (str | Unset):
        resolved_at (datetime.datetime | None | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        tool_input (AgentBridgeApprovalToolInput | Unset):
    """

    bridge_id: str
    created_at: datetime.datetime
    event_name: str
    id: str
    provider: str
    status: str
    tool_name: str
    decision: AgentBridgeHookDecision | Unset = UNSET
    pty_id: str | Unset = UNSET
    resolved_at: datetime.datetime | None | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    tool_input: AgentBridgeApprovalToolInput | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        bridge_id = self.bridge_id

        created_at = self.created_at.isoformat()

        event_name = self.event_name

        id = self.id

        provider = self.provider

        status = self.status

        tool_name = self.tool_name

        decision: dict[str, Any] | Unset = UNSET
        if not isinstance(self.decision, Unset):
            decision = self.decision.to_dict()

        pty_id = self.pty_id

        resolved_at: None | str | Unset
        if isinstance(self.resolved_at, Unset):
            resolved_at = UNSET
        elif isinstance(self.resolved_at, datetime.datetime):
            resolved_at = self.resolved_at.isoformat()
        else:
            resolved_at = self.resolved_at

        run_id = self.run_id

        session_id = self.session_id

        tool_input: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tool_input, Unset):
            tool_input = self.tool_input.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "bridgeId": bridge_id,
                "createdAt": created_at,
                "eventName": event_name,
                "id": id,
                "provider": provider,
                "status": status,
                "toolName": tool_name,
            }
        )
        if decision is not UNSET:
            field_dict["decision"] = decision
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if resolved_at is not UNSET:
            field_dict["resolvedAt"] = resolved_at
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if tool_input is not UNSET:
            field_dict["toolInput"] = tool_input

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_bridge_approval_tool_input import (
            AgentBridgeApprovalToolInput,
        )
        from ..models.agent_bridge_hook_decision import AgentBridgeHookDecision

        d = dict(src_dict)
        bridge_id = d.pop("bridgeId")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        event_name = d.pop("eventName")

        id = d.pop("id")

        provider = d.pop("provider")

        status = d.pop("status")

        tool_name = d.pop("toolName")

        _decision = d.pop("decision", UNSET)
        decision: AgentBridgeHookDecision | Unset
        if isinstance(_decision, Unset):
            decision = UNSET
        else:
            decision = AgentBridgeHookDecision.from_dict(_decision)

        pty_id = d.pop("ptyId", UNSET)

        def _parse_resolved_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                resolved_at_type_0 = datetime.datetime.fromisoformat(data)

                return resolved_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        resolved_at = _parse_resolved_at(d.pop("resolvedAt", UNSET))

        run_id = d.pop("runId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        _tool_input = d.pop("toolInput", UNSET)
        tool_input: AgentBridgeApprovalToolInput | Unset
        if isinstance(_tool_input, Unset):
            tool_input = UNSET
        else:
            tool_input = AgentBridgeApprovalToolInput.from_dict(_tool_input)

        agent_bridge_approval = cls(
            bridge_id=bridge_id,
            created_at=created_at,
            event_name=event_name,
            id=id,
            provider=provider,
            status=status,
            tool_name=tool_name,
            decision=decision,
            pty_id=pty_id,
            resolved_at=resolved_at,
            run_id=run_id,
            session_id=session_id,
            tool_input=tool_input,
        )

        agent_bridge_approval.additional_properties = d
        return agent_bridge_approval

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
