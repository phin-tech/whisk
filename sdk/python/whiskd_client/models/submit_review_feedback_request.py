from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="SubmitReviewFeedbackRequest")


@_attrs_define
class SubmitReviewFeedbackRequest:
    """
    Attributes:
        body (str):
        work_item_id (str):
        actor (str | Unset):
        run_id (str | Unset):
    """

    body: str
    work_item_id: str
    actor: str | Unset = UNSET
    run_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        body = self.body

        work_item_id = self.work_item_id

        actor = self.actor

        run_id = self.run_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "body": body,
                "workItemId": work_item_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if run_id is not UNSET:
            field_dict["runId"] = run_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        work_item_id = d.pop("workItemId")

        actor = d.pop("actor", UNSET)

        run_id = d.pop("runId", UNSET)

        submit_review_feedback_request = cls(
            body=body,
            work_item_id=work_item_id,
            actor=actor,
            run_id=run_id,
        )

        submit_review_feedback_request.additional_properties = d
        return submit_review_feedback_request

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
