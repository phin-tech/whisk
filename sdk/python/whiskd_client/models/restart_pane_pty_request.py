from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.start_pty_options import StartPTYOptions


T = TypeVar("T", bound="RestartPanePTYRequest")


@_attrs_define
class RestartPanePTYRequest:
    """
    Attributes:
        options (StartPTYOptions):
        pane_id (str):
        session_id (str):
    """

    options: StartPTYOptions
    pane_id: str
    session_id: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        options = self.options.to_dict()

        pane_id = self.pane_id

        session_id = self.session_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "options": options,
                "paneId": pane_id,
                "sessionId": session_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.start_pty_options import StartPTYOptions

        d = dict(src_dict)
        options = StartPTYOptions.from_dict(d.pop("options"))

        pane_id = d.pop("paneId")

        session_id = d.pop("sessionId")

        restart_pane_pty_request = cls(
            options=options,
            pane_id=pane_id,
            session_id=session_id,
        )

        restart_pane_pty_request.additional_properties = d
        return restart_pane_pty_request

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
