# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Be sure to restart your server when you modify this file.

# Add new inflection rules using the following format
# (all these examples are active by default):
# ActiveSupport::Inflector.inflections do |inflect|
#   inflect.plural /^(ox)$/i, '\1en'
#   inflect.singular /^(ox)en/i, '\1'
#   inflect.irregular 'person', 'people'
#   inflect.uncountable %w( fish sheep )
# end

ActiveSupport::Inflector.inflections do |inflect|
  inflect.plural(/^([Ss]pecimen)$/i, '\1s')
  inflect.singular(/^([Ss]pecimen)s?/i, '\1')
  inflect.plural(/^([Hh]uman)$/i, '\1s')
  inflect.singular(/^([Hh]uman)s?/i, '\1')
end
