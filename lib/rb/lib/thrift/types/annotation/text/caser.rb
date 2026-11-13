module Thrift
  module Types
    module Annotation
      module Text
        # Caser renders ASCII identifiers in a naming convention: the
        # identifier is split in words, see split_words, each word is cased
        # then the words are joined.
        #
        # It mirrors the internal wordcase package of the go library.
        #
        # @api private
        class Caser
          # The common initialisms of Go identifiers, as listed by revive.
          GO_INITIALISMS = %w[
            ACL API ASCII CPU CSS DNS EOF GUID HTML HTTP HTTPS ID IDs IP JSON
            LHS QPS RAM RHS RPC SLA SMTP SQL SSH TCP TLS TTL UDP UI UID UUID
            URI URL UTF8 VM XML XMPP XSRF XSS
          ].freeze

          class << self
            # Splits the ASCII identifier in words on underscores, on lower to
            # upper case changes and at the end of acronyms: "PaypalAPI" is
            # "Paypal" and "API", "HTTPServer" is "HTTP" and "Server". Digits
            # stick to the preceding word.
            def split_words(str)
              words = []
              start = 0

              str.each_char.with_index do |c, i|
                if c == '_'
                  words << str[start...i] if i > start
                  start = i + 1
                elsif i > start && upper?(c) && word_start?(str, i)
                  words << str[start...i]
                  start = i
                end
              end

              words << str[start..] if start < str.length
              words
            end

            private

            # Reports whether the upper case character at i starts a word: it
            # follows a lower case letter or a digit, or it ends an acronym.
            def word_start?(str, index)
              prev = str[index - 1]

              return true if lower?(prev) || digit?(prev)

              upper?(prev) && index + 1 < str.length && lower?(str[index + 1])
            end

            def upper?(char)
              char.between?('A', 'Z')
            end

            def lower?(char)
              char.between?('a', 'z')
            end

            def digit?(char)
              char.between?('0', '9')
            end
          end

          # Initialisms are rendered as given by CamelCase and PascalCase, e.g.
          # "API" or "IDs". They are matched case insensitively.
          def initialize(kase, initialisms = [])
            @case = kase
            @initialisms = initialisms.to_h { |i| [i.upcase, i] }.freeze
          end

          def call(str)
            return str if @case == Case::Identity

            words = self.class.split_words(str).each_with_index.map { |w, i| case_word(w, i) }

            case @case
            when Case::KebabCase then words.join('-')
            when Case::CamelCase, Case::PascalCase, Case::LowerCase then words.join
            else words.join('_')
            end
          end

          private

          def case_word(word, index)
            case @case
            when Case::ScreamingSnakeCase then word.upcase
            when Case::CamelCase, Case::PascalCase then camel_word(word, index.zero? && @case == Case::CamelCase)
            else word.downcase
            end
          end

          def camel_word(word, first)
            return word.downcase if first

            @initialisms.fetch(word.upcase) { word[0].upcase + word[1..].downcase }
          end
        end
      end
    end
  end
end
