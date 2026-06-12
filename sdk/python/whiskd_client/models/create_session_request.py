from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.start_pty_options import StartPTYOptions


T = TypeVar("T", bound="CreateSessionRequest")


@_attrs_define
class CreateSessionRequest:
    """
    Attributes:
        name (str):
        root_dir (str):
        initial_pty (None | StartPTYOptions | Unset):
    """

    name: str
    root_dir: str
    initial_pty: None | StartPTYOptions | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.start_pty_options import StartPTYOptions

        name = self.name

        root_dir = self.root_dir

        initial_pty: dict[str, Any] | None | Unset
        if isinstance(self.initial_pty, Unset):
            initial_pty = UNSET
        elif isinstance(self.initial_pty, StartPTYOptions):
            initial_pty = self.initial_pty.to_dict()
        else:
            initial_pty = self.initial_pty

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "rootDir": root_dir,
            }
        )
        if initial_pty is not UNSET:
            field_dict["initialPty"] = initial_pty

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.start_pty_options import StartPTYOptions

        d = dict(src_dict)
        name = d.pop("name")

        root_dir = d.pop("rootDir")

        def _parse_initial_pty(data: object) -> None | StartPTYOptions | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                initial_pty_type_1 = StartPTYOptions.from_dict(data)

                return initial_pty_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | StartPTYOptions | Unset, data)

        initial_pty = _parse_initial_pty(d.pop("initialPty", UNSET))

        create_session_request = cls(
            name=name,
            root_dir=root_dir,
            initial_pty=initial_pty,
        )

        create_session_request.additional_properties = d
        return create_session_request

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
