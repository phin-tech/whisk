from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_bridge_event_option import AgentBridgeEventOption
    from ..models.agent_prompt_tool_input import AgentPromptToolInput


T = TypeVar("T", bound="AgentPrompt")


@_attrs_define
class AgentPrompt:
    """
    Attributes:
        created_at (datetime.datetime):
        event_name (str):
        id (str):
        kind (str):
        message (str):
        provider (str):
        status (str):
        answer (str | Unset):
        bridge_id (str | Unset):
        cwd (str | Unset):
        elicitation_id (str | Unset):
        options (list[AgentBridgeEventOption] | Unset):
        pty_id (str | Unset):
        resolved_at (datetime.datetime | None | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        tool_input (AgentPromptToolInput | Unset):
        tool_name (str | Unset):
    """

    created_at: datetime.datetime
    event_name: str
    id: str
    kind: str
    message: str
    provider: str
    status: str
    answer: str | Unset = UNSET
    bridge_id: str | Unset = UNSET
    cwd: str | Unset = UNSET
    elicitation_id: str | Unset = UNSET
    options: list[AgentBridgeEventOption] | Unset = UNSET
    pty_id: str | Unset = UNSET
    resolved_at: datetime.datetime | None | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    tool_input: AgentPromptToolInput | Unset = UNSET
    tool_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        event_name = self.event_name

        id = self.id

        kind = self.kind

        message = self.message

        provider = self.provider

        status = self.status

        answer = self.answer

        bridge_id = self.bridge_id

        cwd = self.cwd

        elicitation_id = self.elicitation_id

        options: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = []
            for options_item_data in self.options:
                options_item = options_item_data.to_dict()
                options.append(options_item)

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

        tool_name = self.tool_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "eventName": event_name,
                "id": id,
                "kind": kind,
                "message": message,
                "provider": provider,
                "status": status,
            }
        )
        if answer is not UNSET:
            field_dict["answer"] = answer
        if bridge_id is not UNSET:
            field_dict["bridgeId"] = bridge_id
        if cwd is not UNSET:
            field_dict["cwd"] = cwd
        if elicitation_id is not UNSET:
            field_dict["elicitationId"] = elicitation_id
        if options is not UNSET:
            field_dict["options"] = options
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
        if tool_name is not UNSET:
            field_dict["toolName"] = tool_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_bridge_event_option import AgentBridgeEventOption
        from ..models.agent_prompt_tool_input import AgentPromptToolInput

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        event_name = d.pop("eventName")

        id = d.pop("id")

        kind = d.pop("kind")

        message = d.pop("message")

        provider = d.pop("provider")

        status = d.pop("status")

        answer = d.pop("answer", UNSET)

        bridge_id = d.pop("bridgeId", UNSET)

        cwd = d.pop("cwd", UNSET)

        elicitation_id = d.pop("elicitationId", UNSET)

        _options = d.pop("options", UNSET)
        options: list[AgentBridgeEventOption] | Unset = UNSET
        if _options is not UNSET:
            options = []
            for options_item_data in _options:
                options_item = AgentBridgeEventOption.from_dict(options_item_data)

                options.append(options_item)

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
        tool_input: AgentPromptToolInput | Unset
        if isinstance(_tool_input, Unset):
            tool_input = UNSET
        else:
            tool_input = AgentPromptToolInput.from_dict(_tool_input)

        tool_name = d.pop("toolName", UNSET)

        agent_prompt = cls(
            created_at=created_at,
            event_name=event_name,
            id=id,
            kind=kind,
            message=message,
            provider=provider,
            status=status,
            answer=answer,
            bridge_id=bridge_id,
            cwd=cwd,
            elicitation_id=elicitation_id,
            options=options,
            pty_id=pty_id,
            resolved_at=resolved_at,
            run_id=run_id,
            session_id=session_id,
            tool_input=tool_input,
            tool_name=tool_name,
        )

        agent_prompt.additional_properties = d
        return agent_prompt

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
