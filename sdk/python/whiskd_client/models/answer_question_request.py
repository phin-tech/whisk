from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AnswerQuestionRequest")


@_attrs_define
class AnswerQuestionRequest:
    """
    Attributes:
        answer (str):
        id (str):
        actor (str | Unset):
    """

    answer: str
    id: str
    actor: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        answer = self.answer

        id = self.id

        actor = self.actor

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "answer": answer,
                "id": id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answer = d.pop("answer")

        id = d.pop("id")

        actor = d.pop("actor", UNSET)

        answer_question_request = cls(
            answer=answer,
            id=id,
            actor=actor,
        )

        answer_question_request.additional_properties = d
        return answer_question_request

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
