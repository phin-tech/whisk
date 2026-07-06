from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AgentStatus")


@_attrs_define
class AgentStatus:
    """
    Attributes:
        advisory (bool):
        agent (str):
        confidence (str):
        label (str):
        source (str):
        state (str):
        prompt (str | Unset):
        title (str | Unset):
    """

    advisory: bool
    agent: str
    confidence: str
    label: str
    source: str
    state: str
    prompt: str | Unset = UNSET
    title: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        advisory = self.advisory

        agent = self.agent

        confidence = self.confidence

        label = self.label

        source = self.source

        state = self.state

        prompt = self.prompt

        title = self.title

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "advisory": advisory,
                "agent": agent,
                "confidence": confidence,
                "label": label,
                "source": source,
                "state": state,
            }
        )
        if prompt is not UNSET:
            field_dict["prompt"] = prompt
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        advisory = d.pop("advisory")

        agent = d.pop("agent")

        confidence = d.pop("confidence")

        label = d.pop("label")

        source = d.pop("source")

        state = d.pop("state")

        prompt = d.pop("prompt", UNSET)

        title = d.pop("title", UNSET)

        agent_status = cls(
            advisory=advisory,
            agent=agent,
            confidence=confidence,
            label=label,
            source=source,
            state=state,
            prompt=prompt,
            title=title,
        )

        agent_status.additional_properties = d
        return agent_status

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
