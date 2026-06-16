from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.agent_bridge_hook_response_output import AgentBridgeHookResponseOutput


T = TypeVar("T", bound="AgentBridgeHookResponse")


@_attrs_define
class AgentBridgeHookResponse:
    """
    Attributes:
        output (AgentBridgeHookResponseOutput | Unset):
    """

    output: AgentBridgeHookResponseOutput | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        output: dict[str, Any] | Unset = UNSET
        if not isinstance(self.output, Unset):
            output = self.output.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if output is not UNSET:
            field_dict["output"] = output

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agent_bridge_hook_response_output import (
            AgentBridgeHookResponseOutput,
        )

        d = dict(src_dict)
        _output = d.pop("output", UNSET)
        output: AgentBridgeHookResponseOutput | Unset
        if isinstance(_output, Unset):
            output = UNSET
        else:
            output = AgentBridgeHookResponseOutput.from_dict(_output)

        agent_bridge_hook_response = cls(
            output=output,
        )

        agent_bridge_hook_response.additional_properties = d
        return agent_bridge_hook_response

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
